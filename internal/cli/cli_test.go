package cli_test

import (
	"bytes"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KrishRVH/pixel-bonsai/internal/cli"
	"github.com/KrishRVH/pixel-bonsai/internal/usage"
)

// run runs a command line in a home and state folder of its own and returns its exit code and output. Each DIR in
// the line becomes a new temporary folder.
func run(t *testing.T, env map[string]string, line string) (code int, stdout, stderr string) {
	t.Helper()
	args := strings.Fields(line)
	for k, a := range args {
		if a == "DIR" {
			args[k] = t.TempDir()
		}
	}
	var out, errs bytes.Buffer
	app := cli.App{
		Stdout: &out,
		Stderr: &errs,
		Getenv: func(k string) string { return env[k] },
		Home:   env["HOME"],
		GOOS:   "linux",
		Now:    func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) },
	}
	code = app.Main(t.Context(), args)
	return code, out.String(), errs.String()
}

func folders(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{"HOME": t.TempDir(), "BONSAI_STATE_DIR": filepath.Join(t.TempDir(), "state")}
}

func TestRenderRunsInsideAnyHostWithoutATerminalOrGarden(t *testing.T) {
	t.Parallel()
	env := folders(t)
	code, out, _ := run(t, env, "render --growth 0.7 --variety maple --format json")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var frame struct {
		Width, Height  int
		Variety, Stage string
		Pixels         [][3]any
	}
	if err := json.Unmarshal([]byte(out), &frame); err != nil {
		t.Fatal(err)
	}
	if frame.Width != 48 || frame.Height != 30 || frame.Variety != "maple" || frame.Stage != "mature" || len(frame.Pixels) == 0 {
		t.Errorf("frame = %d×%d %s %s with %d pixels", frame.Width, frame.Height, frame.Variety, frame.Stage, len(frame.Pixels))
	}
	for _, p := range frame.Pixels {
		x, y, rgb := p[0].(float64), p[1].(float64), p[2].(string)
		if x < 0 || x >= 48 || y < 0 || y >= 30 || len(rgb) != 6 {
			t.Fatalf("pixel %v is off the canvas or not rrggbb", p)
		}
	}
	if _, out, _ := run(t, env, "render --growth 0.7"); !strings.Contains(out, "▀") {
		t.Error("ANSI render has no half blocks")
	}
	_, out, _ = run(t, env, "render --growth 0.7 --format png --zoom 2 --scale 0.5")
	img, err := png.Decode(strings.NewReader(out))
	if err != nil || img.Bounds().Dx() != 48 || img.Bounds().Dy() != 32 {
		t.Errorf("PNG render is %v, %v: want 48×32, a 24×16 frame at zoom 2", img.Bounds(), err)
	}
	if _, err := os.Stat(env["BONSAI_STATE_DIR"]); !os.IsNotExist(err) {
		t.Error("render touched the state folder")
	}
}

func TestUsageErrorsExitTwoAndExplain(t *testing.T) {
	t.Parallel()
	env := folders(t)
	for _, line := range []string{
		"grow",
		"render",
		"render --growth 2",
		"render --growth 0.5 --variety baobab",
		"render --growth 0.5 --format gif",
		"render --growth 0.5 --scale 9",
		"render --growth 0.5 --scale NaN",
		"render --growth NaN",
		"sheet DIR --scale NaN",
		"render --growth 0.5 --color sepia",
		"render --growth 0.5 extra",
		"render --bogus",
		"record --source s --id i",
		"record --source s --id i --tokens -1",
		"record --source s --id i --tokens 1 --timestamp yesterday",
		"sheet",
		"sheet DIR baobab",
		"--color sepia",
	} {
		code, _, errs := run(t, env, line)
		if code != 2 || !strings.HasPrefix(errs, "pixel-bonsai: ") || strings.Count(errs, "\n") != 2 {
			t.Errorf("%q: exit %d, stderr %q; want 2 and an explanation", line, code, errs)
		}
	}
}

func TestHelpAndVersionSucceed(t *testing.T) {
	t.Parallel()
	env := folders(t)
	if code, out, _ := run(t, env, "help"); code != 0 || !strings.Contains(out, "pixel-bonsai record") {
		t.Errorf("help: exit %d, %q", code, out)
	}
	if code, _, errs := run(t, env, "render -h"); code != 0 || !strings.Contains(errs, "-growth") {
		t.Errorf("render -h: exit %d, %q", code, errs)
	}
	if code, out, _ := run(t, env, "version"); code != 0 || !strings.HasPrefix(out, "pixel-bonsai ") {
		t.Errorf("version: exit %d, %q", code, out)
	}
}

func TestRecordFeedsDoctor(t *testing.T) {
	t.Parallel()
	env := folders(t)
	for range 2 {
		if code, _, errs := run(t, env, "record --source my-tool --id turn42 --tokens 71 --timestamp 2026-11-19T10:30:00+02:00"); code != 0 {
			t.Fatalf("record: exit %d, %s", code, errs)
		}
	}
	code, out, errs := run(t, env, "doctor --json")
	if code != 0 {
		t.Fatalf("doctor: exit %d, %s", code, errs)
	}
	var report struct {
		State   string
		Color   string
		Sources []struct {
			Name          string
			Events, Today int64
			Tokens        int64
		}
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if report.State != env["BONSAI_STATE_DIR"] || report.Color != "truecolor" || len(report.Sources) != len(usage.New(usage.Env{}).Ledgers) {
		t.Fatalf("report = %+v", report)
	}
	feed := report.Sources[len(report.Sources)-1]
	if feed.Name != "events" || feed.Events != 1 || feed.Tokens != 71 || feed.Today != 71 {
		t.Errorf("feed = %+v, want one event of 71 tokens today: a retry counts once", feed)
	}
	if code, out, _ := run(t, env, "doctor"); code != 0 || !strings.Contains(out, "events") || !strings.Contains(out, "garden") {
		t.Errorf("doctor: exit %d, %q", code, out)
	}
}

func TestSheetTakesFlagsBeforeOrAfterItsArguments(t *testing.T) {
	t.Parallel()
	env := folders(t)
	for _, line := range []string{"sheet --zoom 1 DIR maple", "sheet DIR --zoom 1 maple", "sheet DIR maple --zoom=1"} {
		code, out, errs := run(t, env, line)
		if files := strings.Fields(out); code != 0 || len(files) != 5 {
			t.Errorf("%q: exit %d, %d files, %s", line, code, len(files), errs)
		}
	}
}

func TestTheLiveTreeNeedsATerminal(t *testing.T) {
	t.Parallel()
	code, _, errs := run(t, folders(t), "")
	if code != 1 || !strings.Contains(errs, "needs a terminal") {
		t.Errorf("exit %d, %q", code, errs)
	}
}
