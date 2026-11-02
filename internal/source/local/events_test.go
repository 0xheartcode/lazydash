package local

import "testing"

// TestListEventsMapsOpLog hand-authors an issue ref whose commit chain carries
// the gli trailer block (Op/Actor/Lamport/Format-Version) and asserts that
// ListEvents maps each commit to the expected Event: friendly kinds, the folded
// title, actor resolution, and newest-first order. It uses only git, no gli.
func TestListEventsMapsOpLog(t *testing.T) {
	dir, tree := newRepo(t)

	createMsg := "Fix login crash\n\nUsers hit a crash on special chars.\n\nOp: create\nActor: alice\nState: open\nLamport: 1\nFormat-Version: 1\n"
	r := commitTree(t, dir, tree, "", createMsg, "2026-01-01T10:00:00")

	commentMsg := "Reproduced on Firefox\n\nHappens on 120.\n\nOp: comment\nActor: bob\nLamport: 2\nFormat-Version: 1\n"
	c := commitTree(t, dir, tree, r, commentMsg, "2026-01-02T10:00:00")

	// A set-state commit with no Actor trailer, so the actor falls back to the
	// git author ("Tester" from the shared commit identity).
	closeMsg := "Close issue\n\nOp: set-state\nState: closed\nLamport: 3\nFormat-Version: 1\n"
	x := commitTree(t, dir, tree, c, closeMsg, "2026-01-03T10:00:00")

	const id = "a7f3b2c1-1111-2222-3333-444455556666"
	runGit(t, dir, nil, "", "update-ref", issueRefPrefix+id, x)

	events, err := ListEvents(dir, map[string]string{id: "Fix login crash"})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	// Newest-first: Closed, Commented, Created.
	wantKind := []string{"Closed", "Commented", "Created"}
	wantActor := []string{"Tester", "bob", "alice"}
	for i, e := range events {
		if e.Kind != wantKind[i] {
			t.Errorf("event[%d].Kind = %q, want %q", i, e.Kind, wantKind[i])
		}
		if e.Actor != wantActor[i] {
			t.Errorf("event[%d].Actor = %q, want %q", i, e.Actor, wantActor[i])
		}
		if e.IssueID != id {
			t.Errorf("event[%d].IssueID = %q, want %q", i, e.IssueID, id)
		}
		if e.IssueTitle != "Fix login crash" {
			t.Errorf("event[%d].IssueTitle = %q, want %q", i, e.IssueTitle, "Fix login crash")
		}
	}

	// Descending committer dates.
	if !(events[0].Date > events[1].Date && events[1].Date > events[2].Date) {
		t.Errorf("events not newest-first: %q %q %q", events[0].Date, events[1].Date, events[2].Date)
	}
}

// TestListEventsFallsBackToSubjectTitle checks that when no title is supplied
// the create commit's subject is used as the issue title.
func TestListEventsFallsBackToSubjectTitle(t *testing.T) {
	dir, tree := newRepo(t)
	msg := "Add dark mode\n\nWould be nice.\n\nOp: create\nActor: carol\nFormat-Version: 1\n"
	r := commitTree(t, dir, tree, "", msg, "2026-02-01T09:00:00")
	runGit(t, dir, nil, "", "update-ref", issueRefPrefix+"bbbb0000", r)

	events, err := ListEvents(dir, nil)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].IssueTitle != "Add dark mode" {
		t.Errorf("IssueTitle = %q, want %q (subject fallback)", events[0].IssueTitle, "Add dark mode")
	}
	if events[0].Kind != "Created" {
		t.Errorf("Kind = %q, want Created", events[0].Kind)
	}
}

// TestEventKindMapping exercises the op-to-label table directly.
func TestEventKindMapping(t *testing.T) {
	cases := []struct {
		op, state, want string
	}{
		{"create", "", "Created"},
		{"comment", "", "Commented"},
		{"set-state", "closed", "Closed"},
		{"set-state", "open", "Reopened"},
		{"add-label", "", "Labeled"},
		{"remove-label", "", "Unlabeled"},
		{"add-assignee", "", "Assigned"},
		{"remove-assignee", "", "Unassigned"},
		{"set-priority", "", "Prioritized"},
		{"set-title", "", "Retitled"},
		{"set-description", "", "Edited"},
		{"link-pr", "", "link-pr"}, // unknown op falls through to its slug
		{"", "", "Updated"},
	}
	for _, tc := range cases {
		tr := map[string]string{"Op": tc.op}
		if tc.state != "" {
			tr["State"] = tc.state
		}
		if got := eventKind(tr); got != tc.want {
			t.Errorf("eventKind(Op=%q,State=%q) = %q, want %q", tc.op, tc.state, got, tc.want)
		}
	}
}
