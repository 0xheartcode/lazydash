package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// git-local-issue (gli) owns the issue format lazydash reads and writes. When the
// gli binary is present, reads fold through its JSON output and writes shell out
// to its subcommands, so lazydash never has to reimplement the on-disk format. A
// git-plumbing fallback (read.go) keeps viewing working when gli is absent.

// gliAvailable reports whether the gli binary is on PATH. Reads fall back to git
// plumbing when it is not, so writes are gated separately from reads.
func gliAvailable() bool {
	_, err := exec.LookPath("gli")
	return err == nil
}

// gli runs a gli command with its working directory set to the repo root and
// returns raw stdout. Stderr is folded into the error so callers get an
// actionable message.
func gli(root string, args ...string) (string, error) {
	cmd := exec.Command("gli", args...)
	cmd.Dir = root
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gli %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// gliIssue mirrors one element of `gli ls --format json`: a fully folded issue.
// Only the fields lazydash renders are decoded; unknown fields are ignored.
type gliIssue struct {
	UUID         string       `json:"uuid"`
	Number       int          `json:"number"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	State        string       `json:"state"`
	Assignee     *string      `json:"assignee"`
	Assignees    []string     `json:"assignees"`
	Priority     string       `json:"priority"`
	Labels       []string     `json:"labels"`
	Archived     bool         `json:"archived"`
	CommentCount int          `json:"comment_count"`
	Comments     []gliComment `json:"comments"`
}

// gliComment mirrors one element of a gli issue's "comments" array.
type gliComment struct {
	Author string `json:"author"`
	Text   string `json:"text"`
}

// parseGliList folds a `gli ls --format json` payload into the shared Issue
// shape. It is kept separate from the binary invocation so it can be exercised
// against canned JSON with no gli binary present.
func parseGliList(data []byte) ([]Issue, error) {
	var raw []gliIssue
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse gli json: %w", err)
	}
	issues := make([]Issue, 0, len(raw))
	for _, g := range raw {
		issues = append(issues, g.toIssue())
	}
	return issues, nil
}

// toIssue maps a decoded gli issue onto the shared Issue shape.
func (g gliIssue) toIssue() Issue {
	assignees := g.Assignees
	// Fall back to the legacy single assignee when the OR-set is empty.
	if len(assignees) == 0 && g.Assignee != nil && strings.TrimSpace(*g.Assignee) != "" {
		assignees = []string{*g.Assignee}
	}

	var comments []Comment
	for _, c := range g.Comments {
		comments = append(comments, Comment{Author: c.Author, Body: c.Text})
	}

	return Issue{
		ID:           g.UUID,
		Number:       g.Number,
		Title:        g.Title,
		Body:         g.Description,
		State:        strings.ToLower(g.State),
		Labels:       g.Labels,
		Assignees:    assignees,
		Priority:     strings.ToLower(g.Priority),
		Archived:     g.Archived,
		CommentCount: g.CommentCount,
		Comments:     comments,
	}
}

// ListIssuesGli reads every issue (including archived ones) by parsing
// `gli ls --all --format json`, run with its working directory at the repo root.
func ListIssuesGli(root string) ([]Issue, error) {
	out, err := gli(root, "ls", "--all", "--format", "json")
	if err != nil {
		return nil, err
	}
	return parseGliList([]byte(out))
}
