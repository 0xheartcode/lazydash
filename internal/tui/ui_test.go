package tui

import (
	"path/filepath"
	"testing"

	"github.com/0xheartcode/lazydash/internal/config"
	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
	tea "github.com/charmbracelet/bubbletea"
)

// fakeSource is a writable in-memory backend for driving the TUI in tests.
type fakeSource struct {
	created []source.Draft
	board   *core.BoardData
}

func (f *fakeSource) Name() string { return "fake" }
func (f *fakeSource) Caps() source.Capabilities {
	return source.Capabilities{Create: true, Comment: true, SetState: true}
}
func (f *fakeSource) ListProjects() ([]core.Project, error) {
	return []core.Project{{ID: "p1", Title: "Fake"}}, nil
}
func (f *fakeSource) GetBoard(string) (*core.BoardData, error) { return f.board, nil }
func (f *fakeSource) CreateIssue(_ string, d source.Draft) (core.Card, error) {
	f.created = append(f.created, d)
	return core.Card{ID: "new", Title: d.Title}, nil
}
func (f *fakeSource) Comment(core.Card, string) error          { return nil }
func (f *fakeSource) SetState(core.Card, string) error         { return nil }
func (f *fakeSource) SetField(core.Card, string, string) error { return nil }
func (f *fakeSource) SetLabels(core.Card, []string) error      { return nil }
func (f *fakeSource) SetAssignees(core.Card, []string) error   { return nil }

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "none.yml"))
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// TestCreateFlowInvokesWriter drives create end to end: press c, type a title,
// press enter, and run the resulting command — asserting the writer was called.
func TestCreateFlowInvokesWriter(t *testing.T) {
	fake := &fakeSource{board: &core.BoardData{Views: []core.ProjectView{{Name: "Board", Layout: "BOARD_LAYOUT"}}}}
	m := newModel(testConfig(t))
	m.registry = source.NewRegistry(fake)
	m.activeProject = core.Project{ID: "p1", Source: "fake"}
	m.active = paneBoard

	// Size the modal.
	upd, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = upd.(Model)

	// Open the create prompt.
	upd, _ = m.Update(key("c"))
	m = upd.(Model)
	if m.mode != modeInput {
		t.Fatalf("after create key, mode = %v, want modeInput", m.mode)
	}

	// Type a title.
	for _, ch := range []string{"F", "i", "x"} {
		upd, _ = m.Update(key(ch))
		m = upd.(Model)
	}

	// Submit.
	upd, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = upd.(Model)
	if m.mode != modeNormal {
		t.Errorf("after submit, mode = %v, want modeNormal", m.mode)
	}
	if cmd == nil {
		t.Fatal("submit produced no command")
	}
	// Draining the batch runs the mutation command.
	drain(cmd)

	if len(fake.created) != 1 {
		t.Fatalf("CreateIssue called %d times, want 1", len(fake.created))
	}
	if fake.created[0].Title != "Fix" {
		t.Errorf("created title = %q, want Fix", fake.created[0].Title)
	}
}

// drain executes a command (and any batched children) to trigger side effects.
func drain(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			drain(c)
		}
	}
}

func TestReadOnlySourceIgnoresCreate(t *testing.T) {
	m := newModel(testConfig(t))
	m.registry = source.NewRegistry(readOnlySource{})
	m.activeProject = core.Project{ID: "p1", Source: "ro"}
	m.active = paneBoard

	upd, _ := m.Update(key("c"))
	m = upd.(Model)
	if m.mode != modeNormal {
		t.Errorf("read-only source should not open the create modal; mode = %v", m.mode)
	}
}

// readOnlySource advertises no write capabilities.
type readOnlySource struct{}

func (readOnlySource) Name() string                             { return "ro" }
func (readOnlySource) Caps() source.Capabilities                { return source.Capabilities{} }
func (readOnlySource) ListProjects() ([]core.Project, error)    { return nil, nil }
func (readOnlySource) GetBoard(string) (*core.BoardData, error) { return &core.BoardData{}, nil }

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" bug ,, auth ,")
	if len(got) != 2 || got[0] != "bug" || got[1] != "auth" {
		t.Errorf("splitCSV = %v, want [bug auth]", got)
	}
}

func TestCanWrite(t *testing.T) {
	if canWrite(source.Capabilities{}) {
		t.Error("empty capabilities should not be writable")
	}
	if !canWrite(source.Capabilities{Comment: true}) {
		t.Error("comment capability should be writable")
	}
}

// subMap extracts a nested snapshot section, failing if it is not a map.
func subMap(t *testing.T, s map[string]any, k string) map[string]any {
	t.Helper()
	v, ok := s[k].(map[string]any)
	if !ok {
		t.Fatalf("snapshot[%q] is not a map (got %T)", k, s[k])
	}
	return v
}

// boardFixture builds a board grouped by Status into three columns holding four
// cards: Todo (Alpha, Bravo), In Progress (Charlie), Done (Delta).
func boardFixture() *core.BoardData {
	return &core.BoardData{
		Views: []core.ProjectView{{Name: "Board", Layout: "BOARD_LAYOUT", GroupByField: "Status"}},
		Fields: []core.SelectField{{
			Name:    "Status",
			Options: []core.FieldOption{{Name: "Todo"}, {Name: "In Progress"}, {Name: "Done"}},
		}},
		Items: []core.RawItem{
			{ID: "1", Type: "ISSUE", Number: 1, Title: "Alpha", State: "open", FieldValues: map[string]string{"Status": "Todo"}},
			{ID: "2", Type: "ISSUE", Number: 2, Title: "Bravo", State: "open", FieldValues: map[string]string{"Status": "Todo"}},
			{ID: "3", Type: "ISSUE", Number: 3, Title: "Charlie", State: "open", FieldValues: map[string]string{"Status": "In Progress"}},
			{ID: "4", Type: "ISSUE", Number: 4, Title: "Delta", State: "open", FieldValues: map[string]string{"Status": "Done"}},
		},
	}
}

// newBoardModel returns a sized Model backed by a writable fake source with the
// board fixture already loaded, leaving the board pane active.
func newBoardModel(t *testing.T) Model {
	t.Helper()
	fake := &fakeSource{board: boardFixture()}
	m := newModel(testConfig(t))
	m.registry = source.NewRegistry(fake)
	m.activeProject = core.Project{ID: "p1", Source: "fake"}

	upd, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = upd.(Model)

	upd, _ = m.Update(boardLoadedMsg{data: fake.board})
	m = upd.(Model)
	return m
}

// TestProjectsLoadedPopulatesList feeds a projectsLoadedMsg and checks the
// project list count and that the projects pane stays active.
func TestProjectsLoadedPopulatesList(t *testing.T) {
	m := newModel(testConfig(t))
	upd, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = upd.(Model)

	projects := []core.Project{
		{ID: "p1", Title: "One", Source: "fake"},
		{ID: "p2", Title: "Two", Source: "fake"},
	}
	upd, _ = m.Update(projectsLoadedMsg{projects: projects})
	m = upd.(Model)

	s := m.snapshot()
	if s["pane"] != "projects" {
		t.Errorf("pane = %v, want projects", s["pane"])
	}
	if got := subMap(t, s, "projects")["count"]; got != 2 {
		t.Errorf("projects.count = %v, want 2", got)
	}
	if s["loading"] != false {
		t.Errorf("loading = %v, want false", s["loading"])
	}
}

// TestBoardLoadedShowsBoard loads a board and checks the column/card counts and
// that the active pane switches to the board.
func TestBoardLoadedShowsBoard(t *testing.T) {
	m := newBoardModel(t)

	s := m.snapshot()
	if s["pane"] != "board" {
		t.Errorf("pane = %v, want board", s["pane"])
	}
	b := subMap(t, s, "board")
	if b["columns"] != 3 {
		t.Errorf("board.columns = %v, want 3", b["columns"])
	}
	if b["cards"] != 4 {
		t.Errorf("board.cards = %v, want 4", b["cards"])
	}
	if b["selected_title"] != "Alpha" {
		t.Errorf("board.selected_title = %v, want Alpha", b["selected_title"])
	}
}

// TestBoardNavigationChangesSelection drives j/k/l/h and asserts the selected
// card title tracks the cursor across rows and columns.
func TestBoardNavigationChangesSelection(t *testing.T) {
	m := newBoardModel(t)

	selected := func() any { return subMap(t, m.snapshot(), "board")["selected_title"] }

	if got := selected(); got != "Alpha" {
		t.Fatalf("initial selected = %v, want Alpha", got)
	}

	steps := []struct {
		press string
		want  string
	}{
		{"j", "Bravo"},   // down within the Todo column
		{"k", "Alpha"},   // back up
		{"l", "Charlie"}, // right into In Progress
		{"l", "Delta"},   // right into Done
		{"h", "Charlie"}, // left back to In Progress
	}
	for _, st := range steps {
		upd, _ := m.Update(key(st.press))
		m = upd.(Model)
		if got := selected(); got != st.want {
			t.Errorf("after %q, selected = %v, want %v", st.press, got, st.want)
		}
	}
}

// TestHelpAndDetailsToggle flips the help overlay and the card detail overlay
// and asserts both flags in the snapshot.
func TestHelpAndDetailsToggle(t *testing.T) {
	m := newBoardModel(t)

	if m.snapshot()["help_open"] != false {
		t.Fatalf("help_open should start false")
	}
	upd, _ := m.Update(key("?"))
	m = upd.(Model)
	if m.snapshot()["help_open"] != true {
		t.Errorf("help_open = false after toggle, want true")
	}
	upd, _ = m.Update(key("?"))
	m = upd.(Model)
	if m.snapshot()["help_open"] != false {
		t.Errorf("help_open = true after second toggle, want false")
	}

	if m.snapshot()["details_open"] != false {
		t.Fatalf("details_open should start false")
	}
	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = upd.(Model)
	if m.snapshot()["details_open"] != true {
		t.Errorf("details_open = false after enter, want true")
	}
	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = upd.(Model)
	if m.snapshot()["details_open"] != false {
		t.Errorf("details_open = true after esc, want false")
	}
}

// TestModalModeTransitions opens the create input modal and the close/reopen
// confirm modal, asserting the snapshot mode and action for each.
func TestModalModeTransitions(t *testing.T) {
	m := newBoardModel(t)

	s := m.snapshot()
	if s["mode"] != "normal" || s["action"] != "none" {
		t.Fatalf("initial mode/action = %v/%v, want normal/none", s["mode"], s["action"])
	}

	// Create opens an input modal.
	upd, _ := m.Update(key("c"))
	m = upd.(Model)
	s = m.snapshot()
	if s["mode"] != "input" {
		t.Errorf("after create, mode = %v, want input", s["mode"])
	}
	if s["action"] != "create" {
		t.Errorf("after create, action = %v, want create", s["action"])
	}

	// Escape returns to normal.
	upd, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = upd.(Model)
	s = m.snapshot()
	if s["mode"] != "normal" || s["action"] != "none" {
		t.Errorf("after esc, mode/action = %v/%v, want normal/none", s["mode"], s["action"])
	}

	// Toggle state on an open card opens a close confirmation.
	upd, _ = m.Update(key("x"))
	m = upd.(Model)
	s = m.snapshot()
	if s["mode"] != "confirm" {
		t.Errorf("after toggle state, mode = %v, want confirm", s["mode"])
	}
	if s["action"] != "close" {
		t.Errorf("after toggle state, action = %v, want close", s["action"])
	}
}
