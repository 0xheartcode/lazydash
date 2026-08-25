package github

import (
	"fmt"

	"github.com/0xheartcode/lazydash/internal/core"
)

// cacheFieldIDs records the field and option ids of a freshly loaded board so a
// subsequent SetField can turn the field/option names the TUI works with back
// into the ids the GitHub API requires.
func (s *Source) cacheFieldIDs(projectID string, bd *core.BoardData) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastProjID = projectID
	s.fieldIDs = make(map[string]string, len(bd.Fields))
	s.optionIDs = make(map[string]map[string]string, len(bd.Fields))
	for _, f := range bd.Fields {
		s.fieldIDs[f.Name] = f.ID
		opts := make(map[string]string, len(f.Options))
		for _, o := range f.Options {
			opts[o.Name] = o.ID
		}
		s.optionIDs[f.Name] = opts
	}
}

// SetField moves a card to a single-select option — the GitHub equivalent of
// dragging it to another board column — via `gh project item-edit`. It resolves
// the project/field/option ids from the cache populated by the last GetBoard.
func (s *Source) SetField(item core.Card, field, option string) error {
	if !s.writable {
		return fmt.Errorf("gh CLI not found — cannot edit project fields")
	}
	s.mu.Lock()
	projID := s.lastProjID
	fieldID := s.fieldIDs[field]
	optionID := s.optionIDs[field][option]
	s.mu.Unlock()

	if projID == "" || fieldID == "" || optionID == "" {
		return fmt.Errorf("cannot resolve ids for field %q option %q", field, option)
	}
	_, err := gh("project", "item-edit",
		"--id", item.ID,
		"--project-id", projID,
		"--field-id", fieldID,
		"--single-select-option-id", optionID,
	)
	return err
}
