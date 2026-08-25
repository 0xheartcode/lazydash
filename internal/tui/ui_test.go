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
