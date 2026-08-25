package github

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// Compile-time proof the GitHub backend implements both interfaces.
var (
	_ source.Source = (*Source)(nil)
	_ source.Writer = (*Source)(nil)
)

// GitHub mutations shell out to the authenticated gh CLI — the tool lazydash
// already requires for auth — rather than hand-rolling GraphQL. gh accepts an
// issue/PR URL directly, so a card's URL is all we need; no node-id lookup.

// ghAvailable reports whether the gh binary is on PATH. Auth via go-gh does not
// require the binary, so writes are gated separately from reads.
func ghAvailable() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func gh(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// ghKind returns the gh subcommand group ("issue" or "pr") for a card, or an
// error for draft issues, which have no gh-addressable object.
func ghKind(item core.Card) (string, error) {
	switch item.Type {
	case "ISSUE":
		return "issue", nil
	case "PULL_REQUEST":
		return "pr", nil
	default:
		return "", fmt.Errorf("draft issues cannot be edited on GitHub")
	}
}

// Comment adds a comment to the underlying issue or pull request.
func (s *Source) Comment(item core.Card, body string) error {
	kind, err := ghKind(item)
	if err != nil {
		return err
	}
	if item.URL == "" {
		return fmt.Errorf("item has no URL to comment on")
	}
	_, err = gh(kind, "comment", item.URL, "--body", body)
	return err
}

// SetState closes or reopens the underlying issue or pull request.
func (s *Source) SetState(item core.Card, state string) error {
	kind, err := ghKind(item)
	if err != nil {
		return err
	}
	if item.URL == "" {
		return fmt.Errorf("item has no URL")
	}
	verb := "reopen"
	if strings.EqualFold(state, "closed") {
		verb = "close"
	}
	_, err = gh(kind, verb, item.URL)
	return err
}

// The remaining mutations are not wired for GitHub yet. They satisfy the Writer
// interface but are gated off via Caps, so the TUI never invokes them; the
// errors are a safety net rather than a user-facing path.

func (s *Source) CreateIssue(string, source.Draft) (core.Card, error) {
	return core.Card{}, fmt.Errorf("creating GitHub issues is not supported yet")
}

func (s *Source) SetField(core.Card, string, string) error {
	return fmt.Errorf("moving GitHub cards is not supported yet")
}

func (s *Source) SetLabels(core.Card, []string) error {
	return fmt.Errorf("editing GitHub labels is not supported yet")
}

func (s *Source) SetAssignees(core.Card, []string) error {
	return fmt.Errorf("editing GitHub assignees is not supported yet")
}
