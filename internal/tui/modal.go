package tui

import (
	"strings"

	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
	"github.com/0xheartcode/lazydash/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) openInput(action modalAction, title, placeholder string, multiline bool) tea.Cmd {
	m.action = action
	m.mode = modeInput
	return m.prompt.Open(title, placeholder, multiline)
}

func (m *Model) openPicker(action modalAction, title string, options []string) {
	m.action = action
	m.mode = modePicker
	m.picker.Open(title, options)
}

func (m *Model) openConfirm(action modalAction, text string) {
	m.action = action
	m.mode = modeConfirm
	m.confirm = text
}

func (m *Model) closeModal() {
	m.mode = modeNormal
	m.action = actNone
	m.confirm = ""
}

func (m Model) startCreate() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Create {
		m.setStatus("create not supported by this source")
		return m, nil
	}
	return m, m.openInput(actCreate, "New issue title", "e.g. Fix login crash", false)
}

func (m Model) startComment() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Comment || m.board.SelectedCard() == nil {
		return m, nil
	}
	return m, m.openInput(actComment, "Comment", "type a comment", true)
}

func (m *Model) startToggleState() {
	if !m.activeCaps().SetState {
		m.setStatus("state changes not supported by this source")
		return
	}
	card := m.board.SelectedCard()
	if card == nil {
		return
	}
	title := utils.Truncate(card.Title, 40)
	if strings.EqualFold(card.State, "closed") {
		m.openConfirm(actReopen, "Reopen \""+title+"\"?")
	} else {
		m.openConfirm(actClose, "Close \""+title+"\"?")
	}
}

func (m *Model) startMove() {
	if !m.activeCaps().SetField || m.board.SelectedCard() == nil {
		return
	}
	field := m.currentGroupField()
	opts := m.fieldOptions(field)
	if field == "" || len(opts) == 0 {
		m.setStatus("no field to move by in this view")
		return
	}
	m.openPicker(actMove, "Move to "+field, opts)
}

func (m Model) startLabels() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Labels || m.board.SelectedCard() == nil {
		return m, nil
	}
	return m, m.openInput(actLabels, "Labels (comma-separated)", "bug, auth", false)
}

func (m Model) startAssign() (tea.Model, tea.Cmd) {
	if !m.activeCaps().Assign || m.board.SelectedCard() == nil {
		return m, nil
	}
	return m, m.openInput(actAssign, "Assignee", "name or email", false)
}

func (m Model) handleModalKey(msg tea.KeyMsg, k string) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInput:
		switch k {
		case "esc":
			m.closeModal()
			return m, nil
		case "enter":
			if !m.prompt.Multiline() {
				return m.submitModal()
			}
		case "ctrl+d":
			if m.prompt.Multiline() {
				return m.submitModal()
			}
		}
		return m, m.prompt.Update(msg)
	case modePicker:
		switch k {
		case "esc":
			m.closeModal()
		case m.keys.Up:
			m.picker.MoveUp()
		case m.keys.Down:
			m.picker.MoveDown()
		case "enter":
			return m.submitModal()
		}
		return m, nil
	case modeConfirm:
		switch k {
		case "y", "enter":
			return m.submitModal()
		case "n", "esc":
			m.closeModal()
		}
		return m, nil
	}
	return m, nil
}

// submitModal dispatches the open modal's action to the active source's Writer
// and returns a command that performs the mutation and reloads the board.
func (m Model) submitModal() (tea.Model, tea.Cmd) {
	action := m.action
	var value string
	switch m.mode {
	case modeInput:
		value = strings.TrimSpace(m.prompt.Value())
	case modePicker:
		value = m.picker.Selected()
	}
	card := m.board.SelectedCard()
	proj := m.activeProject
	m.closeModal()

	w, ok := m.activeWriter()
	if !ok {
		m.setStatus("this source is read-only")
		return m, nil
	}

	run := func(status string, fn func() error) (tea.Model, tea.Cmd) {
		m.loading = true
		m.setStatus(status)
		return m, tea.Batch(m.spinner.Tick, mutateCmd(proj, fn))
	}

	switch action {
	case actCreate:
		if value == "" {
			m.setStatus("create cancelled — empty title")
			return m, nil
		}
		return run("Creating issue…", func() error {
			_, err := w.CreateIssue(proj.ID, source.Draft{Title: value})
			return err
		})
	case actComment:
		if value == "" || card == nil {
			return m, nil
		}
		item := *card
		return run("Adding comment…", func() error { return w.Comment(item, value) })
	case actMove:
		if card == nil || value == "" {
			return m, nil
		}
		item, field := *card, m.currentGroupField()
		return run("Moving…", func() error { return w.SetField(item, field, value) })
	case actLabels:
		if card == nil {
			return m, nil
		}
		item, labels := *card, splitCSV(value)
		return run("Updating labels…", func() error { return w.SetLabels(item, labels) })
	case actAssign:
		if card == nil {
			return m, nil
		}
		item, who := *card, splitCSV(value)
		return run("Updating assignee…", func() error { return w.SetAssignees(item, who) })
	case actClose:
		if card == nil {
			return m, nil
		}
		item := *card
		return run("Closing…", func() error { return w.SetState(item, "closed") })
	case actReopen:
		if card == nil {
			return m, nil
		}
		item := *card
		return run("Reopening…", func() error { return w.SetState(item, "open") })
	}
	return m, nil
}

// currentGroupField is the field the current view groups its columns by.
func (m Model) currentGroupField() string {
	if m.boardData == nil || m.viewIdx < 0 || m.viewIdx >= len(m.boardData.Views) {
		return ""
	}
	return m.boardData.Views[m.viewIdx].GroupByField
}

// fieldOptions returns the option names of a single-select field.
func (m Model) fieldOptions(field string) []string {
	if m.boardData == nil || field == "" {
		return nil
	}
	for _, f := range m.boardData.Fields {
		if f.Name == field {
			opts := make([]string, 0, len(f.Options))
			for _, o := range f.Options {
				opts = append(opts, o.Name)
			}
			return opts
		}
	}
	return nil
}

// mutateCmd runs a write off the UI goroutine and reports the outcome.
func mutateCmd(proj core.Project, fn func() error) tea.Cmd {
	return func() tea.Msg {
		if err := fn(); err != nil {
			return mutationDoneMsg{err: err}
		}
		return mutationDoneMsg{reload: proj}
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
