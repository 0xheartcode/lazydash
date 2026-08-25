package github

import (
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func TestCacheFieldIDsResolvesNames(t *testing.T) {
	s := &Source{}
	bd := &core.BoardData{Fields: []core.SelectField{{
		ID:   "F1",
		Name: "Status",
		Options: []core.FieldOption{
			{ID: "O1", Name: "Todo"},
			{ID: "O2", Name: "Done"},
		},
	}}}
	s.cacheFieldIDs("PROJ", bd)

	if s.lastProjID != "PROJ" {
		t.Errorf("lastProjID = %q, want PROJ", s.lastProjID)
	}
	if s.fieldIDs["Status"] != "F1" {
		t.Errorf("fieldIDs[Status] = %q, want F1", s.fieldIDs["Status"])
	}
	if s.optionIDs["Status"]["Done"] != "O2" {
		t.Errorf("optionIDs[Status][Done] = %q, want O2", s.optionIDs["Status"]["Done"])
	}
}

func TestSetFieldErrorsWhenUnresolved(t *testing.T) {
	// Writable but no board cached yet: SetField must fail before shelling to gh.
	s := &Source{writable: true}
	if err := s.SetField(core.Card{ID: "I1"}, "Status", "Done"); err == nil {
		t.Error("SetField should error when ids are unresolved")
	}
}

func TestSetFieldErrorsWhenNotWritable(t *testing.T) {
	s := &Source{writable: false}
	if err := s.SetField(core.Card{ID: "I1"}, "Status", "Done"); err == nil {
		t.Error("SetField should error when gh is unavailable")
	}
}
