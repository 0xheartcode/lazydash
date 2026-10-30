package sidebar

import (
	"strings"
	"testing"
)

// renderMarkdown should produce styled terminal output (ANSI escapes for
// headings, list bullets, code) rather than falling back to the plain wrapper.
func TestRenderMarkdownStylesContent(t *testing.T) {
	out := renderMarkdown("## Heading\n\n- one **bold** item\n", 60)
	if out == "" {
		t.Fatal("renderMarkdown returned empty output")
	}
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI-styled markdown, got plain text: %q", out)
	}
	// A rendered list turns the "-" marker into a bullet; its presence proves we
	// went through glamour rather than the plain-text fallback.
	if !strings.Contains(out, "•") {
		t.Errorf("expected a rendered list bullet, got: %q", out)
	}
}

func TestRenderMarkdownFallsBackOnZeroWidth(t *testing.T) {
	if got := renderMarkdown("plain body", 0); got != "plain body" {
		t.Errorf("zero width should return the input unchanged, got %q", got)
	}
}
