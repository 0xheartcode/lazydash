package github

import (
	"reflect"
	"testing"

	"github.com/0xheartcode/lazydash/internal/core"
)

func TestMergeOrgs(t *testing.T) {
	tests := []struct {
		name       string
		discovered []string
		configured []string
		want       []string
	}{
		{
			name:       "empty inputs",
			discovered: nil,
			configured: nil,
			want:       nil,
		},
		{
			name:       "discovered only",
			discovered: []string{"acme", "globex"},
			configured: nil,
			want:       []string{"acme", "globex"},
		},
		{
			name:       "configured appended after discovered",
			discovered: []string{"acme"},
			configured: []string{"globex"},
			want:       []string{"acme", "globex"},
		},
		{
			name:       "case-insensitive dedup keeps first casing",
			discovered: []string{"Acme"},
			configured: []string{"acme"},
			want:       []string{"Acme"},
		},
		{
			name:       "dedup within discovered preserves order and first casing",
			discovered: []string{"Acme", "globex", "ACME"},
			configured: nil,
			want:       []string{"Acme", "globex"},
		},
		{
			name:       "configured duplicate of discovered is dropped",
			discovered: []string{"Acme", "Globex"},
			configured: []string{"ACME", "initech"},
			want:       []string{"Acme", "Globex", "initech"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeOrgs(tt.discovered, tt.configured)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeOrgs(%v, %v) = %v, want %v", tt.discovered, tt.configured, got, tt.want)
			}
		})
	}
}

func TestMergeOrgsDoesNotMutateInputs(t *testing.T) {
	discovered := []string{"acme"}
	configured := []string{"globex"}
	mergeOrgs(discovered, configured)
	if !reflect.DeepEqual(discovered, []string{"acme"}) {
		t.Errorf("discovered mutated: %v", discovered)
	}
	if !reflect.DeepEqual(configured, []string{"globex"}) {
		t.Errorf("configured mutated: %v", configured)
	}
}

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		name string
		item string
		list []string
		want bool
	}{
		{name: "empty list", item: "acme", list: nil, want: false},
		{name: "exact match", item: "acme", list: []string{"acme"}, want: true},
		{name: "case-insensitive match", item: "ACME", list: []string{"acme"}, want: true},
		{name: "mixed case in list", item: "acme", list: []string{"Globex", "ACME"}, want: true},
		{name: "no match", item: "initech", list: []string{"acme", "globex"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIgnored(tt.item, tt.list); got != tt.want {
				t.Errorf("isIgnored(%q, %v) = %v, want %v", tt.item, tt.list, got, tt.want)
			}
		})
	}
}

func titles(projects []core.Project) []string {
	out := make([]string, len(projects))
	for i, p := range projects {
		out[i] = p.Title
	}
	return out
}

func TestFilterIgnoredProjects(t *testing.T) {
	projects := []core.Project{
		{Owner: "acme", Title: "Roadmap"},
		{Owner: "globex", Title: "Roadmap"},
		{Owner: "acme", Title: "Backlog"},
	}

	tests := []struct {
		name   string
		ignore []string
		want   []string
	}{
		{
			name:   "empty ignore returns all",
			ignore: nil,
			want:   []string{"Roadmap", "Roadmap", "Backlog"},
		},
		{
			name:   "bare title removes all matches across owners",
			ignore: []string{"Roadmap"},
			want:   []string{"Backlog"},
		},
		{
			name:   "owner/title removes only that owner",
			ignore: []string{"acme/Roadmap"},
			want:   []string{"Roadmap", "Backlog"},
		},
		{
			name:   "case-insensitive owner/title",
			ignore: []string{"ACME/roadmap"},
			want:   []string{"Roadmap", "Backlog"},
		},
		{
			name:   "case-insensitive bare title",
			ignore: []string{"BACKLOG"},
			want:   []string{"Roadmap", "Roadmap"},
		},
		{
			name:   "no match returns all",
			ignore: []string{"nonexistent"},
			want:   []string{"Roadmap", "Roadmap", "Backlog"},
		},
		{
			name:   "multiple ignore entries combine",
			ignore: []string{"acme/Roadmap", "Backlog"},
			want:   []string{"Roadmap"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(filterIgnoredProjects(projects, tt.ignore))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterIgnoredProjects titles = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterIgnoredProjectsEmptyReturnsInput(t *testing.T) {
	projects := []core.Project{{Owner: "acme", Title: "Roadmap"}}
	got := filterIgnoredProjects(projects, nil)
	if !reflect.DeepEqual(got, projects) {
		t.Errorf("empty ignore should return input unchanged, got %v", got)
	}
}

func TestFilterAllowedProjects(t *testing.T) {
	projects := []core.Project{
		{Owner: "acme", Title: "Roadmap"},
		{Owner: "globex", Title: "Roadmap"},
		{Owner: "acme", Title: "Backlog"},
	}

	tests := []struct {
		name string
		only []string
		want []string
	}{
		{
			name: "empty only returns all",
			only: nil,
			want: []string{"Roadmap", "Roadmap", "Backlog"},
		},
		{
			name: "bare title keeps all matches across owners",
			only: []string{"Roadmap"},
			want: []string{"Roadmap", "Roadmap"},
		},
		{
			name: "owner/title keeps only that owner",
			only: []string{"acme/Roadmap"},
			want: []string{"Roadmap"},
		},
		{
			name: "case-insensitive owner/title",
			only: []string{"ACME/roadmap"},
			want: []string{"Roadmap"},
		},
		{
			name: "case-insensitive bare title",
			only: []string{"BACKLOG"},
			want: []string{"Backlog"},
		},
		{
			name: "no match keeps none",
			only: []string{"nonexistent"},
			want: []string{},
		},
		{
			name: "multiple only entries combine without duplicating",
			only: []string{"acme/Roadmap", "Backlog"},
			want: []string{"Roadmap", "Backlog"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titles(filterAllowedProjects(projects, tt.only))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterAllowedProjects titles = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterAllowedProjectsEmptyReturnsInput(t *testing.T) {
	projects := []core.Project{{Owner: "acme", Title: "Roadmap"}}
	got := filterAllowedProjects(projects, nil)
	if !reflect.DeepEqual(got, projects) {
		t.Errorf("empty only should return input unchanged, got %v", got)
	}
}
