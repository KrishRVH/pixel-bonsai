// Command pixel-bonsai grows a pixel-art bonsai in the terminal from the token usage of coding agents.
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop() // the first signal asks the command to stop; a second one ends the process at once
	}()
	home, _ := os.UserHomeDir() // a missing home folder leaves only the state folder's own settings to find files by
	app := cli.App{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Getenv: os.Getenv,
		Home:   home,
		GOOS:   runtime.GOOS,
		Now:    time.Now,
	}
	code := app.Main(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
