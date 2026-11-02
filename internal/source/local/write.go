package local

import (
	"fmt"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// Compile-time proof the local backend implements the read and write interfaces.
// It deliberately does not implement source.Undoer: gli owns the append-only
// operation chain, so lazydash does not reach past it with a raw ref reset.
var (
	_ source.Source = (*Source)(nil)
	_ source.Writer = (*Source)(nil)
)

// Local mutations shell out to gli, the tool that owns the on-disk issue format.
// Each write is one gli invocation run at the repo root; gli accepts either the
// issue UUID (what the reader stamps onto a card) or a short id as the <ID>.

// CreateIssue mints a new issue with `gli create` and returns a card keyed by the
// short id gli prints. The board is reloaded after any write, so the exact handle
// on the returned card is not load-bearing beyond Title and State.
func (s *Source) CreateIssue(_ string, d source.Draft) (core.Card, error) {
	title := strings.TrimSpace(d.Title)
	if title == "" {
		return core.Card{}, fmt.Errorf("issue title is required")
	}
	args := []string{"create", title, "-q"}
	if body := strings.TrimSpace(d.Body); body != "" {
		args = append(args, "-d", body)
	}
	for _, l := range d.Labels {
		if l = strings.TrimSpace(l); l != "" {
			args = append(args, "-l", l)
		}
	}
	if p := strings.TrimSpace(d.Priority); p != "" {
		args = append(args, "-p", p)
	}
	// gli create sets a single assignee; any further assignees are added below.
	var extraAssignees []string
	for i, a := range d.Assignees {
		if a = strings.TrimSpace(a); a == "" {
			continue
		}
		if i == 0 {
			args = append(args, "-a", a)
		} else {
			extraAssignees = append(extraAssignees, a)
		}
	}

	out, err := gli(s.root, args...)
	if err != nil {
		return core.Card{}, err
	}
	id := strings.TrimSpace(out)

	for _, a := range extraAssignees {
		if _, err := gli(s.root, "edit", id, "--add-assignee", a); err != nil {
			return core.Card{}, err
		}
	}
	return core.Card{ID: id, Type: "ISSUE", Title: title, State: "OPEN"}, nil
}

// Comment appends a comment with `gli comment`.
func (s *Source) Comment(item core.Card, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("empty comment")
	}
	id, err := issueID(item)
	if err != nil {
		return err
	}
	_, err = gli(s.root, "comment", id, body)
	return err
}

// SetState closes or reopens an issue with `gli close` / `gli reopen`.
func (s *Source) SetState(item core.Card, state string) error {
	id, err := issueID(item)
	if err != nil {
		return err
	}
	verb := "reopen"
	if strings.EqualFold(strings.TrimSpace(state), "closed") {
		verb = "close"
	}
	_, err = gli(s.root, verb, id)
	return err
}

// SetField sets a single-select field. State maps to close/reopen and Priority
// maps to `gli edit -p`; any other field is a no-op, since gli has no general
// single-select surface to express it.
func (s *Source) SetField(item core.Card, field, option string) error {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "state":
		return s.SetState(item, option)
	case "priority":
		id, err := issueID(item)
		if err != nil {
			return err
		}
		_, err = gli(s.root, "edit", id, "-p", option)
		return err
	default:
		return nil
	}
}

// SetLabels reconciles the issue's labels with the desired set. gli has no
// wholesale "set labels", so the difference against the card's current labels is
// issued as --add-label / --remove-label edits.
func (s *Source) SetLabels(item core.Card, labels []string) error {
	id, err := issueID(item)
	if err != nil {
		return err
	}
	current := make([]string, 0, len(item.Labels))
	for _, l := range item.Labels {
		current = append(current, l.Name)
	}
	add, remove := diffSets(current, labels)
	var args []string
	for _, l := range add {
		args = append(args, "--add-label", l)
	}
	for _, l := range remove {
		args = append(args, "--remove-label", l)
	}
	if len(args) == 0 {
		return nil
	}
	_, err = gli(s.root, append([]string{"edit", id}, args...)...)
	return err
}

// SetAssignees reconciles the issue's assignees with the desired set, issuing the
// --add-assignee / --remove-assignee edits for the difference against the card's
// current assignees.
func (s *Source) SetAssignees(item core.Card, who []string) error {
	id, err := issueID(item)
	if err != nil {
		return err
	}
	add, remove := diffSets(item.Assignees, who)
	var args []string
	for _, a := range add {
		args = append(args, "--add-assignee", a)
	}
	for _, a := range remove {
		args = append(args, "--remove-assignee", a)
	}
	if len(args) == 0 {
		return nil
	}
	_, err = gli(s.root, append([]string{"edit", id}, args...)...)
	return err
}

// issueID returns the gli id for a card (its stamped UUID), or an error when the
// card carries no id to address.
func issueID(item core.Card) (string, error) {
	id := strings.TrimSpace(item.ID)
	if id == "" {
		return "", fmt.Errorf("item has no id")
	}
	return id, nil
}

// diffSets returns the items to add (in want, not in have) and to remove (in
// have, not in want), comparing trimmed non-empty values.
func diffSets(have, want []string) (add, remove []string) {
	haveSet := toSet(have)
	wantSet := toSet(want)
	for v := range wantSet {
		if !haveSet[v] {
			add = append(add, v)
		}
	}
	for v := range haveSet {
		if !wantSet[v] {
			remove = append(remove, v)
		}
	}
	return add, remove
}

// toSet collects trimmed, non-empty values into a set.
func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, v := range items {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = true
		}
	}
	return set
}
