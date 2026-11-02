package local

import (
	"sort"
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
)

// ListEvents reads the operation log of every issue under refs/issues/ in the
// repo at root and returns one core.Event per commit, newest-first. It uses only
// git plumbing (no gli binary), so the Activity and Calendar views work offline.
//
// titles maps an issue id to its current folded title; when an id is absent the
// create commit's subject is used as a fallback, so the reader is usable on its
// own. A malformed ref is skipped rather than failing the whole listing.
func ListEvents(root string, titles map[string]string) ([]core.Event, error) {
	out, err := git(root, "for-each-ref", "--format=%(refname)", issueRefPrefix)
	if err != nil {
		return nil, err
	}
	var events []core.Event
	for _, ref := range strings.Split(strings.TrimSpace(out), "\n") {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		id := strings.TrimPrefix(ref, issueRefPrefix)
		evs, err := readEvents(root, ref, id, titles[id])
		if err != nil {
			continue
		}
		events = append(events, evs...)
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Date > events[j].Date
	})
	return events, nil
}

// readEvents walks one issue ref's commit chain (root to tip) and maps each
// commit to an event. When title is empty it falls back to the create commit's
// subject.
func readEvents(root, ref, id, title string) ([]core.Event, error) {
	format := "%cI" + fieldSep + "%s" + fieldSep + "%an" + fieldSep + "%B"
	out, err := git(root, "log", "--reverse", "-z", "--format="+format, ref)
	if err != nil {
		return nil, err
	}
	var events []core.Event
	baseTitle := ""
	for _, rec := range strings.Split(out, recordSep) {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		parts := strings.SplitN(rec, fieldSep, 4)
		if len(parts) < 4 {
			continue
		}
		date, subject, author := parts[0], parts[1], parts[2]
		_, trailers := parseMessage(parts[3])
		if baseTitle == "" {
			baseTitle = subject
		}
		actor := trailers["Actor"]
		if actor == "" {
			actor = author
		}
		events = append(events, core.Event{
			IssueID:    id,
			IssueTitle: title,
			Actor:      actor,
			Kind:       eventKind(trailers),
			Date:       date,
		})
	}
	if title == "" {
		for i := range events {
			events[i].IssueTitle = baseTitle
		}
	}
	return events, nil
}

// eventKind turns a commit's gli trailers into a friendly operation label,
// driven by the Op trailer (and the State trailer for state changes). An
// unrecognised op falls through to its own slug.
func eventKind(trailers map[string]string) string {
	switch trailers["Op"] {
	case "create":
		return "Created"
	case "comment":
		return "Commented"
	case "set-state":
		switch strings.ToLower(trailers["State"]) {
		case "closed":
			return "Closed"
		case "open":
			return "Reopened"
		default:
			return "State changed"
		}
	case "add-label":
		return "Labeled"
	case "remove-label":
		return "Unlabeled"
	case "add-assignee":
		return "Assigned"
	case "remove-assignee":
		return "Unassigned"
	case "set-priority":
		return "Prioritized"
	case "set-title":
		return "Retitled"
	case "set-description":
		return "Edited"
	case "":
		return "Updated"
	default:
		return trailers["Op"]
	}
}
