package local

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// formatVersion tags issues we create with the git-native-issue format version.
const formatVersion = "1"

// Compile-time proof the local backend implements both interfaces.
var (
	_ source.Source = (*Source)(nil)
	_ source.Writer = (*Source)(nil)
)

// The local backend authors the same commit format its reader parses, so writes
// need no external binary and work fully offline. Each mutation is one commit
// under refs/issues/<id>: a create commit roots a new chain, everything else is
// appended onto the current tip with a compare-and-swap ref update.

// CreateIssue roots a new issue chain under a fresh refs/issues/<uuid>.
func (s *Source) CreateIssue(_ string, d source.Draft) (core.Card, error) {
	if strings.TrimSpace(d.Title) == "" {
		return core.Card{}, fmt.Errorf("issue title is required")
	}
	id, err := newUUID()
	if err != nil {
		return core.Card{}, err
	}
	tree, err := s.emptyTree()
	if err != nil {
		return core.Card{}, err
	}
	commit, err := gitInput(s.root, createMessage(d), "commit-tree", tree)
	if err != nil {
		return core.Card{}, err
	}
	if _, err := git(s.root, "update-ref", issueRefPrefix+id, strings.TrimSpace(commit)); err != nil {
		return core.Card{}, err
	}
	return core.Card{ID: id, Type: "ISSUE", Title: d.Title, State: "OPEN"}, nil
}

// Comment appends a comment event.
func (s *Source) Comment(item core.Card, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("empty comment")
	}
	return s.appendEvent(item.ID, body, nil)
}

// SetState closes or reopens an issue by appending a state event.
func (s *Source) SetState(item core.Card, state string) error {
	state = strings.ToLower(strings.TrimSpace(state))
	subject := "Reopen issue"
	if state == "closed" {
		subject = "Close issue"
	}
	return s.appendEvent(item.ID, subject, map[string]string{"State": state})
}

// SetField sets a single-select field. State maps to close/reopen; every other
// field is recorded as a trailer of the same (canonicalised) name.
func (s *Source) SetField(item core.Card, field, option string) error {
	if strings.EqualFold(field, "State") {
		return s.SetState(item, option)
	}
	key := canonicalTrailerKey(field)
	return s.appendEvent(item.ID, "Set "+key+": "+option, map[string]string{key: option})
}

// SetLabels replaces the label set.
func (s *Source) SetLabels(item core.Card, labels []string) error {
	return s.appendEvent(item.ID, "Update labels", map[string]string{"Labels": strings.Join(labels, ", ")})
}

// SetAssignees replaces the assignee.
func (s *Source) SetAssignees(item core.Card, who []string) error {
	return s.appendEvent(item.ID, "Update assignee", map[string]string{"Assignee": strings.Join(who, ", ")})
}

// appendEvent commits a new event (subject plus optional trailers) onto the tip
// of an issue's chain, using a compare-and-swap ref update so a concurrent write
// is rejected rather than silently lost.
func (s *Source) appendEvent(id, subject string, trailers map[string]string) error {
	ref := issueRefPrefix + id
	tip, err := git(s.root, "rev-parse", "--verify", ref)
	if err != nil {
		return fmt.Errorf("issue %s not found", id)
	}
	tip = strings.TrimSpace(tip)
	tree, err := s.emptyTree()
	if err != nil {
		return err
	}
	commit, err := gitInput(s.root, buildMessage(subject, "", trailers), "commit-tree", tree, "-p", tip)
	if err != nil {
		return err
	}
	_, err = git(s.root, "update-ref", ref, strings.TrimSpace(commit), tip)
	return err
}

// emptyTree returns the repo's empty tree object, writing it if needed.
func (s *Source) emptyTree() (string, error) {
	out, err := gitInput(s.root, "", "mktree")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// createMessage builds the root commit message for a new issue.
func createMessage(d source.Draft) string {
	trailers := map[string]string{"State": "open", "Format-Version": formatVersion}
	if len(d.Labels) > 0 {
		trailers["Labels"] = strings.Join(d.Labels, ", ")
	}
	if d.Priority != "" {
		trailers["Priority"] = d.Priority
	}
	if len(d.Assignees) > 0 {
		trailers["Assignee"] = strings.Join(d.Assignees, ", ")
	}
	return buildMessage(d.Title, d.Body, trailers)
}

// buildMessage assembles a commit message: subject, optional body, then the
// trailer block in a stable key order.
func buildMessage(subject, body string, trailers map[string]string) string {
	var b strings.Builder
	b.WriteString(subject)
	if body = strings.TrimSpace(body); body != "" {
		b.WriteString("\n\n")
		b.WriteString(body)
	}
	order := []string{"State", "Priority", "Labels", "Assignee", "Milestone", "Fixed-By", "Format-Version"}
	var lines []string
	for _, k := range order {
		if v := strings.TrimSpace(trailers[k]); v != "" {
			lines = append(lines, k+": "+v)
		}
	}
	if len(lines) > 0 {
		b.WriteString("\n\n")
		b.WriteString(strings.Join(lines, "\n"))
	}
	b.WriteString("\n")
	return b.String()
}

// canonicalTrailerKey maps a field name to its trailer key spelling.
func canonicalTrailerKey(field string) string {
	switch strings.ToLower(field) {
	case "priority":
		return "Priority"
	case "labels", "label":
		return "Labels"
	case "assignee", "assignees":
		return "Assignee"
	case "milestone":
		return "Milestone"
	case "state":
		return "State"
	default:
		if field == "" {
			return field
		}
		return strings.ToUpper(field[:1]) + field[1:]
	}
}

// newUUID returns a random RFC 4122 version-4 UUID, used to name issue refs.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
