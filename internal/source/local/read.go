package local

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const issueRefPrefix = "refs/issues/"

// Field and record separators for the `git log` stream we parse. Unit Separator
// (0x1f) delimits fields and NUL (via `git log -z`) delimits commits; both are
// control characters that never appear in issue prose.
const (
	fieldSep  = "\x1f"
	recordSep = "\x00"
)

// Issue is a single git-native-issue issue, reconstructed by folding its commit
// chain (create → comments → state changes) into the current state.
type Issue struct {
	ID        string // UUID: the ref name under refs/issues/
	Title     string
	Body      string
	State     string // "open" | "closed"
	Labels    []string
	Assignee  string
	Priority  string
	Milestone string
	Comments  []Comment
	CreatedAt string // ISO-8601 date of the create commit
	UpdatedAt string // ISO-8601 date of the tip commit
}

// Comment is one comment event in an issue's history.
type Comment struct {
	Author string
	Date   string
	Body   string
}

// RepoRoot returns the working-tree root of the git repo containing dir, or an
// error if dir is not inside a repository.
func RepoRoot(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HasIssues reports whether the repo at root has any refs/issues/* refs.
func HasIssues(root string) bool {
	out, err := git(root, "for-each-ref", "--format=%(refname)", issueRefPrefix)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// ListIssues reads and reconstructs every issue under refs/issues/ in the repo
// at root, most-recently-updated first. A malformed ref is skipped rather than
// failing the whole listing.
func ListIssues(root string) ([]Issue, error) {
	out, err := git(root, "for-each-ref", "--format=%(refname)", issueRefPrefix)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	for _, ref := range strings.Split(strings.TrimSpace(out), "\n") {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		iss, err := readIssue(root, ref)
		if err != nil {
			continue
		}
		issues = append(issues, iss)
	}
	sort.SliceStable(issues, func(i, j int) bool {
		return issues[i].UpdatedAt > issues[j].UpdatedAt
	})
	return issues, nil
}

// readIssue walks one issue ref's commit chain (root → tip) and folds the events
// into the issue's current state.
func readIssue(root, ref string) (Issue, error) {
	id := strings.TrimPrefix(ref, issueRefPrefix)
	format := "%H" + fieldSep + "%s" + fieldSep + "%aI" + fieldSep + "%an" + fieldSep + "%B"
	out, err := git(root, "log", "--reverse", "-z", "--format="+format, ref)
	if err != nil {
		return Issue{}, err
	}
	commits := parseCommits(out)
	if len(commits) == 0 {
		return Issue{}, fmt.Errorf("issue %s has no commits", id)
	}

	base := commits[0]
	iss := Issue{
		ID:        id,
		State:     "open",
		Title:     base.subject,
		Body:      base.description,
		CreatedAt: base.date,
		UpdatedAt: commits[len(commits)-1].date,
	}

	// Resolve current metadata by walking tip → root: the nearest event that set
	// a field wins, so a later state change or relabel overrides the create.
	var haveState, haveLabels, havePriority, haveAssignee, haveMilestone bool
	for i := len(commits) - 1; i >= 0; i-- {
		t := commits[i].trailers
		if !haveState {
			if v, ok := t["State"]; ok {
				iss.State = strings.ToLower(v)
				haveState = true
			}
		}
		if !haveLabels {
			if v, ok := t["Labels"]; ok {
				iss.Labels = splitList(v)
				haveLabels = true
			}
		}
		if !havePriority {
			if v, ok := t["Priority"]; ok {
				iss.Priority = strings.ToLower(v)
				havePriority = true
			}
		}
		if !haveAssignee {
			if v, ok := t["Assignee"]; ok {
				iss.Assignee = v
				haveAssignee = true
			}
		}
		if !haveMilestone {
			if v, ok := t["Milestone"]; ok {
				iss.Milestone = v
				haveMilestone = true
			}
		}
	}

	// Comments are the non-root events that are not pure state transitions.
	for _, c := range commits[1:] {
		if _, isState := c.trailers["State"]; isState {
			continue
		}
		body := c.subject
		if c.description != "" {
			body = strings.TrimSpace(c.subject + "\n" + c.description)
		}
		if body == "" {
			continue
		}
		iss.Comments = append(iss.Comments, Comment{Author: c.author, Date: c.date, Body: body})
	}
	return iss, nil
}

// commit is one parsed event in an issue chain.
type commit struct {
	hash        string
	subject     string
	date        string
	author      string
	description string            // body with subject and trailer block removed
	trailers    map[string]string // last value wins within a commit
}

// parseCommits splits the NUL-delimited `git log` stream into commits.
func parseCommits(out string) []commit {
	var commits []commit
	for _, rec := range strings.Split(out, recordSep) {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 5)
		if len(parts) < 5 {
			continue
		}
		c := commit{hash: parts[0], subject: parts[1], date: parts[2], author: parts[3]}
		c.description, c.trailers = parseMessage(parts[4])
		commits = append(commits, c)
	}
	return commits
}

var trailerLine = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9-]*):[ \t]*(.*)$`)

// parseMessage splits a raw commit message (%B, which begins with the subject
// line) into its description — the prose after the subject, minus the trailing
// metadata block — and its trailers.
func parseMessage(raw string) (description string, trailers map[string]string) {
	trailers = map[string]string{}
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 {
		lines = lines[1:] // drop the subject line
	}
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// Find the trailing contiguous block of "Key: value" lines.
	end := len(lines)
	start := end
	for i := end - 1; i >= 0; i-- {
		if trailerLine.MatchString(lines[i]) {
			start = i
			continue
		}
		break
	}
	for _, l := range lines[start:end] {
		if m := trailerLine.FindStringSubmatch(l); m != nil {
			trailers[m[1]] = strings.TrimSpace(m[2])
		}
	}
	description = strings.TrimSpace(strings.Join(lines[:start], "\n"))
	return description, trailers
}

// splitList splits a comma-separated trailer value into trimmed, non-empty items.
func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
