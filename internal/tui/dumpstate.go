package tui

import (
	"encoding/json"
	"os"
)

// This file implements an optional machine-readable state seam for headless
// testing. When --dump-state <path> is set, the model writes a JSON snapshot of
// its settled state on every render, so an external test driver can assert
// against real state rather than parsing the rendered screen. The seam is off by
// default and costs nothing when no path is set.

var snapshotPath string

// SetSnapshotPath enables the state seam, writing snapshots to p on each render.
func SetSnapshotPath(p string) { snapshotPath = p }

// snapshot returns machine-first state: stable keys, enums as strings, counts for
// lists. It is a projection of the model, not a mirror of the rendered layout.
func (m Model) snapshot() map[string]any {
	s := map[string]any{
		"pane":         paneName(m.active),
		"view":         layoutName(m.board.Layout()),
		"loading":      m.loading,
		"error":        m.err != nil,
		"status":       m.status,
		"help_open":    m.showHelp,
		"details_open": m.showDetails,
		"mode":         modeName(m.mode),
		"action":       actionName(m.action),
	}
	if m.err != nil {
		s["error_msg"] = m.err.Error()
	}

	proj := map[string]any{
		"count":  m.projects.Count(),
		"cursor": m.projects.Cursor(),
	}
	if sel := m.projects.Selected(); sel != nil {
		proj["selected"] = sel.Title
		proj["source"] = sel.Source
	}
	s["projects"] = proj

	cols, cards := m.board.Counts()
	b := map[string]any{
		"columns":  cols,
		"cards":    cards,
		"view_idx": m.viewIdx,
		"project":  m.activeProject.Title,
	}
	if sel := m.board.SelectedCard(); sel != nil {
		b["selected_title"] = sel.Title
		b["selected_number"] = sel.Number
		b["selected_state"] = sel.State
		b["selected_status"] = sel.Status
	}
	b["events"] = m.board.EventCount()
	if ev := m.board.SelectedEvent(); ev != nil {
		b["selected_event"] = ev.IssueTitle
	}
	if day := m.board.SelectedDay(); day != "" {
		b["selected_day"] = day
	}
	s["board"] = b

	return s
}

// writeSnapshot atomically writes the model's snapshot to snapshotPath. Best
// effort: a write failure must never disturb the UI, so errors are swallowed.
func writeSnapshot(m Model) {
	if snapshotPath == "" {
		return
	}
	data, err := json.MarshalIndent(m.snapshot(), "", "  ")
	if err != nil {
		return
	}
	tmp := snapshotPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, snapshotPath)
}

func paneName(p pane) string {
	switch p {
	case paneProjects:
		return "projects"
	case paneBoard:
		return "board"
	default:
		return "unknown"
	}
}

func layoutName(layout string) string {
	switch layout {
	case "TABLE_LAYOUT":
		return "table"
	case "ACTIVITY_LAYOUT":
		return "activity"
	case "CALENDAR_LAYOUT":
		return "calendar"
	case "BOARD_LAYOUT":
		return "board"
	default:
		return "board"
	}
}

func modeName(mode modalMode) string {
	switch mode {
	case modeInput:
		return "input"
	case modePicker:
		return "picker"
	case modeConfirm:
		return "confirm"
	default:
		return "normal"
	}
}

func actionName(a modalAction) string {
	switch a {
	case actCreate:
		return "create"
	case actComment:
		return "comment"
	case actMove:
		return "move"
	case actLabels:
		return "labels"
	case actAssign:
		return "assign"
	case actClose:
		return "close"
	case actReopen:
		return "reopen"
	default:
		return "none"
	}
}
