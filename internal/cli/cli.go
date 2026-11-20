// Package cli is the pixel-bonsai command: the live pane and the render, record, sheet, doctor, and version
// subcommands. Everything it touches outside the process comes in through [App], so tests run it against temporary
// folders.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/state"
	"github.com/KrishRVH/pixel-bonsai/internal/usage"
)

// Exit codes: success, a failure, and a usage error.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// An App is the command's connection to its process: standard streams, the environment, the operating system, and
// the clock.
type App struct {
	Stdin          *os.File
	Stdout, Stderr io.Writer
	Getenv         func(string) string
	Home           string // the user's home folder, or "" when it is unknown
	GOOS           string
	Now            func() time.Time
}

const usageText = `Grow a pixel bonsai in your terminal from your coding agents' token usage.

Usage:
  pixel-bonsai [--color MODE]       show the live tree (q quits)
  pixel-bonsai render [flags]       print one frame for another program
  pixel-bonsai record [flags]       record one model call's tokens from any harness
  pixel-bonsai sheet DIR [variety]  write PNG contact sheets of every stage
  pixel-bonsai doctor [--json]      show where usage is read from and what it counts
  pixel-bonsai version              print the version

Run pixel-bonsai COMMAND -h for a command's flags.
`

// Main runs the command with its arguments, not including the program name, and returns the exit code.
func (a App) Main(ctx context.Context, args []string) int {
	cmd, rest := "", args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, rest = args[0], args[1:]
	}
	var err error
	switch cmd {
	case "":
		err = a.live(ctx, rest)
	case "render":
		err = a.render(rest)
	case "record":
		err = a.record(rest)
	case "sheet":
		err = a.sheet(rest)
	case "doctor":
		err = a.doctor(ctx, rest)
	case "version":
		_, err = fmt.Fprintln(a.Stdout, "pixel-bonsai", version())
	case "help":
		_, err = io.WriteString(a.Stdout, usageText)
	default:
		err = usageError{fmt.Errorf("unknown command %q", cmd)}
	}
	return a.exit(err)
}

// A usageError is a mistake in the command line.
type usageError struct{ error }

func (e usageError) Unwrap() error { return e.error }

func (a App) exit(err error) int {
	var bad usageError
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return exitOK
	case errors.As(err, &bad):
		a.warn("pixel-bonsai: %v\nRun pixel-bonsai help for usage, or pixel-bonsai COMMAND -h for a command's flags.\n", err)
		return exitUsage
	}
	a.warn("pixel-bonsai: %v\n", err)
	return exitFail
}

// warn writes to standard error. A failed write there has nowhere left to be reported, so it is dropped.
func (a App) warn(format string, args ...any) {
	_, _ = fmt.Fprintf(a.Stderr, format, args...)
}

// flags returns a flag set whose Usage prints the command's help. Parsing prints nothing itself: parse prints the
// help on -h, and a mistake comes back as a usage error.
func (a App) flags(name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		a.warn("Usage: %s\n\nFlags:\n", synopsis)
		fs.SetOutput(a.Stderr)
		fs.PrintDefaults()
		fs.SetOutput(io.Discard)
	}
	return fs
}

// parse parses flags that may come before, between, or after the positional arguments, and returns those.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	help := fs.Usage
	fs.Usage = func() {} // the flag package calls it on every error; only -h shows the help
	defer func() { fs.Usage = help }()
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				help()
				return nil, err
			}
			return nil, usageError{err}
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// parseNone parses the flags of a command that takes no positional arguments.
func parseNone(fs *flag.FlagSet, args []string) error {
	positional, err := parse(fs, args)
	if err == nil && len(positional) > 0 {
		return usageError{fmt.Errorf("%s takes no arguments, but got %q", fs.Name(), positional)}
	}
	return err
}

func (a App) stateDir() (string, error) {
	return state.Dir(a.Home, a.Getenv, a.GOOS)
}

func (a App) usageEnv() (usage.Env, error) {
	dir, err := a.stateDir()
	if err != nil {
		return usage.Env{}, err
	}
	return usage.Env{Home: a.Home, Getenv: a.Getenv, State: dir}, nil
}

// version returns the module version the binary was built from and its commit, when the build recorded them.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	commit := ""
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 12 {
			commit = " (" + s.Value[:12] + ")"
		}
	}
	return info.Main.Version + commit + " " + info.GoVersion
}
