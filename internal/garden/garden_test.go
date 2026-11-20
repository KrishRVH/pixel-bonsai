package garden_test

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/KrishRVH/pixel-bonsai/bonsai"
	"github.com/KrishRVH/pixel-bonsai/internal/garden"
	"github.com/KrishRVH/pixel-bonsai/internal/state"
)

const (
	day    = "2026-11-19"
	sakura = "sakura"
	maple  = "maple"
	pine   = "black-pine"
	past   = "2026-11-10" // a UTC date with usage
	later  = "2026-11-11"
	claude = "claude"
	baobab = "baobab" // a variety this build does not know
)

// withHistory returns a new garden whose sources have read 100 tokens.
func withHistory() garden.Garden {
	g := garden.New()
	g.Record(map[string]map[string]int64{claude: {past: 100}})
	return g
}

func TestANewGardenGivesTheFirstVarietyEverythingAlreadyCounted(t *testing.T) {
	t.Parallel()
	g := garden.New()
	if g.Variety != sakura || g.View != bonsai.Today || g.Total() != 0 {
		t.Fatalf("new garden = %+v", g)
	}
	g.Record(map[string]map[string]int64{claude: {past: 12e9}, "codex": {later: 1e9}})
	g.Accrue(0, day)
	if g.Grown[sakura] != 13e9 || g.Counted != 13e9 {
		t.Errorf("grown %v, counted %d: the old usage is not sakura's", g.Grown, g.Counted)
	}
}

func TestDailyHighsOnlyRise(t *testing.T) {
	t.Parallel()
	g := garden.New()
	if !g.Record(map[string]map[string]int64{claude: {past: 100, later: 5}}) {
		t.Error("new counts are not a change")
	}
	if g.Record(map[string]map[string]int64{claude: {past: 70}, "codex": {}}) {
		t.Error("a pruned log's smaller count is a change")
	}
	g.Record(map[string]map[string]int64{claude: {later: 9}, "events": {later: 3}})
	if g.Total() != 112 {
		t.Errorf("total = %d, want 112: each source's highest count for each day", g.Total())
	}
	if _, ok := g.History["codex"]; ok {
		t.Error("a source with no usage is kept")
	}
}

func TestAccrueGivesTheSelectedVarietyOnlyWhatIsNew(t *testing.T) {
	t.Parallel()
	g := withHistory()
	g.Accrue(0, day)
	g.Choose(maple)
	g.Record(map[string]map[string]int64{claude: {past: 130}})
	if !g.Accrue(0, day) {
		t.Error("new tokens are not a change")
	}
	g.Choose(pine)
	g.Record(map[string]map[string]int64{claude: {past: 150}})
	g.Accrue(0, day)
	if want := map[string]int64{sakura: 100, maple: 30, pine: 20}; !maps.Equal(g.Grown, want) {
		t.Errorf("grown = %v, want %v", g.Grown, want)
	}
	if g.Accrue(0, day) {
		t.Error("the same total twice is a change")
	}
}

func TestChoosingHandsNothingOut(t *testing.T) {
	t.Parallel()
	g := withHistory()
	g.Accrue(0, day)
	before := g.Clone()
	if !g.Choose(maple) || g.Variety != maple {
		t.Fatal("maple is not chosen")
	}
	if !maps.Equal(g.Grown, before.Grown) || g.Counted != before.Counted {
		t.Error("choosing changed the growth")
	}
	if g.Choose(maple) || g.Choose(baobab) || g.Variety != maple {
		t.Error("choosing the same or an unknown variety is a change")
	}
}

func TestTheDayTreeGrowsOnlyWhileItsVarietyIsSelected(t *testing.T) {
	t.Parallel()
	g := withHistory()
	g.Choose(maple)
	g.Accrue(40, day)
	if g.Day.Grown[maple] != 40 {
		t.Errorf("day = %+v: a new day gives the selected variety all of today so far", g.Day)
	}
	g.Choose(pine)
	g.Accrue(70, day)
	g.Accrue(60, day) // a smaller day count hands nothing out
	got := []int64{g.Tokens(sakura, bonsai.Today, day), g.Tokens(maple, bonsai.Today, day), g.Tokens(pine, bonsai.Today, day)}
	if got[0] != 0 || got[1] != 40 || got[2] != 30 {
		t.Errorf("today's shares = %v, want [0 40 30]", got)
	}
	if g.Tokens(maple, bonsai.Lifetime, day) != 100 {
		t.Errorf("maple's lifetime = %d, want 100", g.Tokens(maple, bonsai.Lifetime, day))
	}
}

func TestTheDayTreeStartsAfreshAtLocalMidnight(t *testing.T) {
	t.Parallel()
	g := withHistory()
	g.Choose(maple)
	g.Accrue(90, "2026-11-18")
	if g.Tokens(maple, bonsai.Today, day) != 0 {
		t.Error("yesterday's record shows once the date has turned")
	}
	if !g.Accrue(5, day) || g.Day.Date != day || g.Day.Counted != 5 || !maps.Equal(g.Day.Grown, map[string]int64{maple: 5}) {
		t.Errorf("day = %+v, want today's 5 tokens alone", g.Day)
	}
	if g.Grown[maple] != 100 {
		t.Error("the lifetime tree lost tokens at midnight")
	}
}

func TestTheGardenSurvivesARestartAndKeepsUnknownVarieties(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "garden.json"), []byte(`{"schema":1,"variety":"maple","view":"lifetime","grown":{"baobab":7},"counted":7}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := garden.Open(dir).Update(func(g *garden.Garden) bool {
		g.Record(map[string]map[string]int64{claude: {past: 160}})
		return g.Accrue(0, day)
	}); err != nil {
		t.Fatal(err)
	}
	g, err := garden.Open(dir).Load()
	if err != nil {
		t.Fatal(err)
	}
	if g.Variety != maple || g.View != bonsai.Lifetime || g.Grown[maple] != 153 || g.Grown[baobab] != 7 {
		t.Errorf("reloaded %+v: want maple's 153 tokens, the lifetime view, and baobab kept", g)
	}
}

func TestAnUpdateWaitsForTheGardenLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := garden.Open(dir)
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		_ = state.Locked(filepath.Join(dir, "garden.json.lock"), func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	go func() {
		_, _ = store.Update(func(g *garden.Garden) bool { return g.Choose(maple) })
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("the update did not wait for the lock")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the update never took the released lock")
	}
	if g, _ := garden.Open(dir).Load(); g.Variety != maple {
		t.Error("the update was lost")
	}
}

func TestADamagedGardenIsSetAsideAndAnUnwritableOneKeepsGoing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "garden.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if g, err := garden.Open(dir).Load(); err == nil || g.Variety != sakura {
		t.Errorf("load of a damaged garden = %+v, %v: want a new garden and the error", g, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("a load moved the damaged garden")
	}
	g, err := garden.Open(dir).Update(func(g *garden.Garden) bool { return g.Choose(maple) })
	if err != nil || g.Variety != maple {
		t.Fatalf("update of a damaged garden = %+v, %v", g, err)
	}
	if data, _ := os.ReadFile(path + ".damaged"); string(data) != "{not json" { // #nosec G304 -- A test's own temporary file.
		t.Error("the damaged file was not set aside")
	}

	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	store := garden.Open(blocker) // a file where the folder should be
	g, err = store.Update(func(g *garden.Garden) bool {
		g.Record(map[string]map[string]int64{claude: {past: 700}})
		return g.Accrue(0, day)
	})
	if err == nil || g.Grown[sakura] != 700 {
		t.Fatalf("unwritable update = %v, %v: want the growth and an error", g.Grown, err)
	}
	g, _ = store.Update(func(g *garden.Garden) bool { return g.Choose(maple) })
	if g.Variety != maple || g.Grown[sakura] != 700 {
		t.Errorf("the next update lost the garden kept in memory: %+v", g)
	}
}

func TestAFailedReadNeverOverwritesTheGarden(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a file this user cannot read")
	}
	dir := t.TempDir()
	store := garden.Open(dir)
	saved, err := store.Update(func(g *garden.Garden) bool {
		return g.Record(map[string]map[string]int64{claude: {past: 5000}}) && g.Accrue(0, day)
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "garden.json")
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(g *garden.Garden) bool { return g.Choose(maple) }); err == nil {
		t.Fatal("an update of an unreadable garden succeeded")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(func(*garden.Garden) bool { return false }); err != nil {
		t.Fatal(err)
	}
	g, err := garden.Open(dir).Load()
	if err != nil || g.Total() != 5000 || g.Grown[saved.Variety] != 5000 {
		t.Errorf("after a failed read the garden is %+v, %v; want the 5000 tokens saved before", g, err)
	}
}

func TestAGardenFromANewerBuildIsLeftAlone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "garden.json")
	newer := []byte(`{"schema":99,"variety":"redwood"}`)
	if err := os.WriteFile(path, newer, 0o600); err != nil {
		t.Fatal(err)
	}
	store := garden.Open(dir)
	if _, err := store.Load(); !errors.Is(err, garden.ErrNewer) {
		t.Errorf("load error = %v, want ErrNewer", err)
	}
	if _, err := store.Update(func(g *garden.Garden) bool { return g.Choose(maple) }); !errors.Is(err, garden.ErrNewer) {
		t.Errorf("update error = %v, want ErrNewer", err)
	}
	if data, _ := os.ReadFile(path); !bytes.Equal(data, newer) { // #nosec G304 -- A test's own temporary file.
		t.Error("the newer garden was overwritten")
	}
}

func TestAHandEditedGardenIsRepaired(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "garden.json"), []byte(`{"variety":"baobab","view":"sideways"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := garden.Open(dir).Load()
	if err != nil || g.Variety != sakura || g.View != bonsai.Today || g.Grown == nil || g.History == nil || g.Day.Grown == nil {
		t.Errorf("repaired garden = %+v, %v", g, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "garden.json.damaged")); !os.IsNotExist(err) {
		t.Error("a hand-edited garden was set aside as damaged")
	}
}
