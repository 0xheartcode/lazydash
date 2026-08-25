// Package source defines the backend abstraction lazydash renders through.
// A Source lists projects and loads their boards; a Source that can also mutate
// issues additionally implements Writer. The TUI talks only to these interfaces
// and the core model, so adding a backend (GitHub Projects, local git issues,
// another forge) never touches the rendering layer.
package source

import "github.com/0xheartcode/lazydash/internal/core"

// Capabilities describes what a source supports. Reads are always available;
// the write flags gate mutation actions in the TUI and correspond to the
// optional Writer methods. A backend advertises only what it can actually do,
// so the same key can be a no-op on one source and a live edit on another.
type Capabilities struct {
	Offline  bool // works with no network (e.g. local git refs)
	Create   bool // create new issues
	Comment  bool // add comments
	SetState bool // close / reopen
	SetField bool // set a single-select field value (move a card between columns)
	Labels   bool // add / remove labels
	Assign   bool // set assignees
}

// Source is a read backend: it lists projects and loads a project's board.
type Source interface {
	// Name is the stable identifier stamped onto core.Project.Source and used
	// to route board loads and mutations back to the owning backend.
	Name() string
	// Caps reports what this source supports right now.
	Caps() Capabilities
	// ListProjects returns the projects this source exposes.
	ListProjects() ([]core.Project, error)
	// GetBoard loads views, fields and items for one project.
	GetBoard(projectID string) (*core.BoardData, error)
}

// Draft is the payload for creating a new issue.
type Draft struct {
	Title     string
	Body      string
	Labels    []string
	Assignees []string
	Priority  string
}

// Writer is implemented by sources that can mutate issues. The TUI type-asserts
// a Source to Writer (via AsWriter) and gates each action on the matching
// Capabilities flag, so a read-only source simply never exposes the action.
//
// Item mutations take the whole core.Card rather than a bare id: the local
// backend keys off Card.ID (the issue UUID) while GitHub keys off Card.URL,
// and passing the card lets each backend read what it needs without a lookup.
type Writer interface {
	CreateIssue(projectID string, d Draft) (core.Card, error)
	Comment(item core.Card, body string) error
	SetState(item core.Card, state string) error // "open" | "closed"
	SetField(item core.Card, field, option string) error
	SetLabels(item core.Card, labels []string) error
	SetAssignees(item core.Card, who []string) error
}

// AsWriter returns the Source as a Writer if it implements one.
func AsWriter(s Source) (Writer, bool) {
	w, ok := s.(Writer)
	return w, ok
}
