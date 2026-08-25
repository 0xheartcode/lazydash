// Package core holds lazydash's backend-neutral domain model: the projects,
// boards, cards and fields that every source (GitHub Projects, local git issues)
// maps onto and that the TUI renders. It knows nothing about where the data
// came from or how it was fetched — that lives in internal/source/*.
package core

// Project represents a board-bearing project from some source.
type Project struct {
	ID          string
	Number      int
	Title       string
	Owner       string
	Description string
	URL         string
	UpdatedAt   string
	Source      string // name of the source that owns this project (e.g. "github", "local")
}

// Column is a named group of cards on a board view.
type Column struct {
	Name  string
	Cards []Card
}

// Label holds an issue label with its display color.
type Label struct {
	Name  string
	Color string // hex color without '#', e.g. "d73a4a"
}

// Card is a single item on the board.
type Card struct {
	ID          string
	Type        string // ISSUE | PULL_REQUEST | DRAFT_ISSUE
	Number      int
	Title       string
	State       string
	URL         string
	Repo        string
	Assignees   []string
	Labels      []Label
	Status      string
	Body        string
	FieldValues map[string]string // all field values keyed by field name
}

// VisibleField is a field configured to be shown in a view.
type VisibleField struct {
	ID       string
	Name     string
	DataType string // TITLE | ASSIGNEES | SINGLE_SELECT | DATE | NUMBER | TEXT | ITERATION | MILESTONE | REPOSITORY | LABELS | LINKED_PULL_REQUESTS
}

// ProjectView mirrors a saved view — a layout plus its grouping and columns.
type ProjectView struct {
	ID            string
	Name          string
	Layout        string // BOARD_LAYOUT | TABLE_LAYOUT | ROADMAP_LAYOUT
	GroupByField  string
	VisibleFields []VisibleField
}

// FieldOption is one option of a single-select field, including its color enum.
type FieldOption struct {
	Name  string
	Color string // enum: GRAY | BLUE | GREEN | YELLOW | ORANGE | RED | PINK | PURPLE
}

// SelectField is a single-select field with ordered options.
type SelectField struct {
	ID      string
	Name    string
	Options []FieldOption
}

// RawItem is an ungrouped project item with all its field values.
type RawItem struct {
	ID          string
	Type        string
	IsArchived  bool
	Number      int
	Title       string
	State       string
	URL         string
	Repo        string
	Assignees   []string
	Labels      []Label
	Body        string
	FieldValues map[string]string // field name → selected option name
}

// BoardData holds everything fetched from a project in one call.
// Grouping into columns is done client-side via GroupByField.
type BoardData struct {
	Views  []ProjectView
	Fields []SelectField
	Items  []RawItem
}
