package tui

import (
	"strings"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func TestPagerContentBuildsMarkdown(t *testing.T) {
	got := pagerContent(&core.Card{Title: "Fix login", Body: "It crashes on submit"})
	if !strings.HasPrefix(got, "# Fix login\n\n") {
		t.Errorf("title should render as an H1 heading, got %q", got)
	}
	if !strings.Contains(got, "It crashes on submit") {
		t.Errorf("body missing from pager content: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("content should end with a newline, got %q", got)
	}
}

func TestPagerContentEmptyCard(t *testing.T) {
	if got := pagerContent(&core.Card{}); got != "" {
		t.Errorf("empty card should yield empty content, got %q", got)
	}
}

func TestResolvePagerHonorsPagerEnv(t *testing.T) {
	t.Setenv("PAGER", "less -R -F")
	name, args := resolvePager()
	if name != "less" {
		t.Errorf("name = %q, want less", name)
	}
	if strings.Join(args, " ") != "-R -F" {
		t.Errorf("args = %v, want [-R -F]", args)
	}
}

func TestResolvePagerFallbackWhenPagerUnset(t *testing.T) {
	t.Setenv("PAGER", "")
	name, _ := resolvePager()
	// bat or less depending on the host; both are acceptable, but never empty.
	if name == "" {
		t.Fatal("resolvePager returned an empty command")
	}
}
