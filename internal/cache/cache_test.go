package cache

import (
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	bd := &core.BoardData{
		Views: []core.ProjectView{{Name: "Board", Layout: "BOARD_LAYOUT"}},
		Items: []core.RawItem{{ID: "1", Title: "A", State: "OPEN"}},
	}
	if err := Save("github/PROJ", bd); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok := Load("github/PROJ")
	if !ok {
		t.Fatal("Load: not found after Save")
	}
	if len(got.Items) != 1 || got.Items[0].Title != "A" {
		t.Errorf("Load returned %+v", got.Items)
	}
	if len(got.Views) != 1 || got.Views[0].Name != "Board" {
		t.Errorf("views = %+v", got.Views)
	}
}

func TestLoadMissingIsNotFound(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if _, ok := Load("never/saved"); ok {
		t.Error("Load of a missing key should report not-found")
	}
}
