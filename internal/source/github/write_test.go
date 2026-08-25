package github

import (
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func TestGhKind(t *testing.T) {
	if k, err := ghKind(core.Card{Type: "ISSUE"}); err != nil || k != "issue" {
		t.Errorf("ISSUE -> (%q, %v), want (issue, nil)", k, err)
	}
	if k, err := ghKind(core.Card{Type: "PULL_REQUEST"}); err != nil || k != "pr" {
		t.Errorf("PULL_REQUEST -> (%q, %v), want (pr, nil)", k, err)
	}
	if _, err := ghKind(core.Card{Type: "DRAFT_ISSUE"}); err == nil {
		t.Error("DRAFT_ISSUE should error — it has no gh-addressable object")
	}
}
