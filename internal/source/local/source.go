package local

import (
	"path/filepath"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// localProjectID is the id of the single synthetic project this source exposes.
const localProjectID = "local"

// Source is the offline git-native-issue backend for one repository. It presents
// the repo's issues as a single project with synthesized fields and views, so
// the board/table renderer treats them exactly like a GitHub project.
type Source struct {
	root string // repo working-tree root
	name string // display name (repo basename)
}

// New builds a local Source for the repo containing dir. It returns (nil, nil)
// when dir is not inside a git repo or the repo has no refs/issues/ — an absent
// local backend is a normal condition (most repos have no local issues), not an
// error that should be surfaced to the user.
func New(dir string) (*Source, error) {
	root, err := RepoRoot(dir)
	if err != nil {
		return nil, nil
	}
	if !HasIssues(root) {
		return nil, nil
	}
	return &Source{root: root, name: filepath.Base(root)}, nil
}

// Name identifies this backend.
func (s *Source) Name() string { return "local" }

// Caps reports the backend's capabilities. It works fully offline and supports
// the whole mutation surface, since writes are authored with git plumbing and
// need no external binary.
func (s *Source) Caps() source.Capabilities {
	return source.Capabilities{
		Offline:  true,
		Create:   true,
		Comment:  true,
		SetState: true,
		SetField: true,
		Labels:   true,
		Assign:   true,
	}
}

// ListProjects exposes the repository as a single synthetic project.
func (s *Source) ListProjects() ([]core.Project, error) {
	return []core.Project{{
		ID:          localProjectID,
		Title:       s.name + " · local",
		Description: "git-native issues in this repository",
	}}, nil
}

// GetBoard reads the repo's issues and folds them into a board with synthesized
// State/Priority fields and Board/Priority/Table views.
func (s *Source) GetBoard(string) (*core.BoardData, error) {
	issues, err := ListIssues(s.root)
	if err != nil {
		return nil, err
	}
	return buildBoard(issues), nil
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

	var assignees []string
	if iss.Assignee != "" {
		assignees = []string{iss.Assignee}
	}

	var labels []core.Label
	for _, name := range iss.Labels {
		labels = append(labels, core.Label{Name: name, Color: labelColor(name)})
	}

	return core.RawItem{
		ID:          iss.ID,
		Type:        "ISSUE",
		Title:       iss.Title,
		State:       strings.ToUpper(iss.State), // OPEN/CLOSED drives the item icon
		Assignees:   assignees,
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
		b.WriteString("— " + who + ":\n" + c.Body)
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
