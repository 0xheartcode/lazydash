package local

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// localRootKey stamps the owning repository root onto each item's FieldValues so
// a write can route back to the right repo. It is not a visible field, so it is
// never rendered; only paths are carried, never anything about the repo's code.
const localRootKey = "__local_root"

// repo is one repository in the multi-repo set.
type repo struct {
	root string // repo working-tree root
	name string // display name (repo basename)
}

// Source is the offline git-native-issue backend. It spans one or more
// repositories, presenting each as its own synthetic project with synthesized
// fields and views, so the board/table renderer treats them exactly like a
// GitHub project. The GitHub backend is already account-wide; this lets the
// local backend span several repos the same way.
type Source struct {
	repos    []repo // discovered repositories, in display order (current repo first)
	writable bool   // gli binary present, so mutations can be shelled out
}

// New builds a local Source for just the repo containing dir. It returns
// (nil, nil) when dir is not inside a git repo or the repo has no refs/issues/:
// an absent local backend is a normal condition (most repos have no local
// issues), not an error that should be surfaced to the user.
func New(dir string) (*Source, error) {
	return NewMulti(dir, nil)
}

// NewMulti builds a local Source spanning the repo containing dir plus each
// extra repository path (from config and the add-repo registry). It returns
// (nil, nil) when no repository resolves, so an empty local backend is simply
// absent rather than an error.
func NewMulti(dir string, extra []string) (*Source, error) {
	refs := Discover(dir, extra)
	if len(refs) == 0 {
		return nil, nil
	}
	rs := make([]repo, len(refs))
	for i, r := range refs {
		rs[i] = repo{root: r.Root, name: r.Name}
	}
	return &Source{repos: rs, writable: gliAvailable()}, nil
}

// Name identifies this backend.
func (s *Source) Name() string { return "local" }

// Caps reports the backend's capabilities. Reads always work offline (via the
// git-plumbing fallback), but every mutation shells out to gli, so the write
// flags are gated on the binary being present.
func (s *Source) Caps() source.Capabilities {
	return source.Capabilities{
		Offline:  true,
		Create:   s.writable,
		Comment:  s.writable,
		SetState: s.writable,
		SetField: s.writable,
		Labels:   s.writable,
		Assign:   s.writable,
	}
}

// ListProjects exposes each repository as its own synthetic project, keyed by
// its working-tree root so GetBoard can route back to the right repo.
func (s *Source) ListProjects() ([]core.Project, error) {
	projects := make([]core.Project, 0, len(s.repos))
	for _, r := range s.repos {
		projects = append(projects, core.Project{
			ID:          r.root,
			Title:       r.name + " · local",
			Description: "git-native issues in " + r.name,
		})
	}
	return projects, nil
}

// GetBoard reads the issues of the repo identified by projectID and folds them
// into a board with synthesized State/Priority fields and Board/Priority/Table
// views. When the gli binary is present the folded state comes from its JSON
// output; otherwise it falls back to the git-plumbing reader so viewing still
// works with no binary.
func (s *Source) GetBoard(projectID string) (*core.BoardData, error) {
	root := s.rootFor(projectID)
	if root == "" {
		return nil, fmt.Errorf("local: unknown project %q", projectID)
	}
	issues, err := s.listIssues(root)
	if err != nil {
		return nil, err
	}
	board := buildBoard(issues)

	// Stamp the owning repo root on each item so writes route to the right repo.
	for i := range board.Items {
		if board.Items[i].FieldValues == nil {
			board.Items[i].FieldValues = map[string]string{}
		}
		board.Items[i].FieldValues[localRootKey] = root
	}

	// The Activity and Calendar views read the raw git op-log directly, keyed to
	// the folded titles we already have. A failure here is non-fatal: the board
	// still renders, those two views are just empty.
	titles := make(map[string]string, len(issues))
	for _, iss := range issues {
		titles[iss.ID] = iss.Title
	}
	if events, err := ListEvents(root, titles); err == nil {
		board.Events = events
	}
	return board, nil
}

// rootFor maps a project id back to a repository root. An empty or legacy
// "local" id maps to the first repo, so older callers and single-repo setups
// keep working.
func (s *Source) rootFor(projectID string) string {
	for _, r := range s.repos {
		if r.root == projectID {
			return r.root
		}
	}
	if (projectID == "" || projectID == "local") && len(s.repos) > 0 {
		return s.repos[0].root
	}
	return ""
}

// rootForCard returns the repository root a card belongs to, read from the stamp
// GetBoard placed on it. It falls back to the first repo for a single-repo setup
// or an unstamped card.
func (s *Source) rootForCard(item core.Card) string {
	if item.FieldValues != nil {
		if r := item.FieldValues[localRootKey]; r != "" {
			return r
		}
	}
	if len(s.repos) > 0 {
		return s.repos[0].root
	}
	return ""
}

// listIssues folds a repo's issues through gli when it is available, and through
// the git-plumbing reader otherwise. Both paths yield the same Issue shape.
func (s *Source) listIssues(root string) ([]Issue, error) {
	if gliAvailable() {
		return ListIssuesGli(root)
	}
	return ListIssues(root)
}

// buildBoard maps reconstructed issues onto the core board model. git-native-issue
// has no notion of projects, fields or views, so we synthesize them: State and
// Priority become single-select fields (giving grouped board columns), and three
// views mirror how you'd browse a GitHub project.
func buildBoard(issues []Issue) *core.BoardData {
	stateField := core.SelectField{Name: "State", Options: []core.FieldOption{
		{Name: "open", Color: "GREEN"},
		{Name: "closed", Color: "GRAY"},
	}}
	priorityField := core.SelectField{Name: "Priority", Options: []core.FieldOption{
		{Name: "critical", Color: "RED"},
		{Name: "high", Color: "ORANGE"},
		{Name: "medium", Color: "YELLOW"},
		{Name: "low", Color: "GRAY"},
	}}

	views := []core.ProjectView{
		{Name: "Board", Layout: "BOARD_LAYOUT", GroupByField: "State", VisibleFields: tableFields()},
		{Name: "By Priority", Layout: "BOARD_LAYOUT", GroupByField: "Priority", VisibleFields: tableFields()},
		{Name: "Table", Layout: "TABLE_LAYOUT", VisibleFields: tableFields()},
		{Name: "Activity", Layout: "ACTIVITY_LAYOUT"},
		{Name: "Calendar", Layout: "CALENDAR_LAYOUT"},
	}

	items := make([]core.RawItem, 0, len(issues))
	for _, iss := range issues {
		items = append(items, issueToItem(iss))
	}
	return &core.BoardData{
		Views:  views,
		Fields: []core.SelectField{stateField, priorityField},
		Items:  items,
	}
}

// tableFields are the columns shown in the table view and used for badges.
func tableFields() []core.VisibleField {
	return []core.VisibleField{
		{Name: "Title", DataType: "TITLE"},
		{Name: "State", DataType: "SINGLE_SELECT"},
		{Name: "Priority", DataType: "SINGLE_SELECT"},
		{Name: "Labels", DataType: "LABELS"},
		{Name: "Assignee", DataType: "ASSIGNEES"},
		{Name: "Milestone", DataType: "TEXT"},
	}
}

// issueToItem converts one reconstructed issue into a board item.
func issueToItem(iss Issue) core.RawItem {
	fv := map[string]string{"State": iss.State}
	if iss.Priority != "" {
		fv["Priority"] = iss.Priority
	}
	if iss.Milestone != "" {
		fv["Milestone"] = iss.Milestone
	}

	var labels []core.Label
	for _, name := range iss.Labels {
		labels = append(labels, core.Label{Name: name, Color: labelColor(name)})
	}

	return core.RawItem{
		ID:          iss.ID,
		Number:      iss.Number,
		Type:        "ISSUE",
		Title:       iss.Title,
		State:       strings.ToUpper(iss.State), // OPEN/CLOSED drives the item icon
		IsArchived:  iss.Archived,
		Assignees:   append([]string(nil), iss.Assignees...),
		Labels:      labels,
		Body:        issueBody(iss),
		FieldValues: fv,
	}
}

// issueBody renders the description followed by any comments, for the sidebar.
func issueBody(iss Issue) string {
	var b strings.Builder
	b.WriteString(iss.Body)
	for _, c := range iss.Comments {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		who := c.Author
		if who == "" {
			who = "comment"
		}
		b.WriteString(who + ":\n" + c.Body)
	}
	return b.String()
}

// labelColor assigns a stable display color to a label name, since
// git-native-issue labels carry no color of their own.
func labelColor(name string) string {
	palette := []string{"d73a4a", "0e8a16", "1d76db", "b60205", "5319e7", "fbca04", "0052cc", "0e6b8a"}
	var sum int
	for _, r := range name {
		sum += int(r)
	}
	return palette[sum%len(palette)]
}
