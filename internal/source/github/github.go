// Package github adapts the GitHub Projects v2 client (internal/api) to the
// source.Source interface: it resolves the viewer, aggregates user and org
// projects, applies the config include/exclude filters, and loads boards.
package github

import (
	"strings"
	"sync"

	"github.com/0xheartcode/lazydash/internal/api"
	"github.com/0xheartcode/lazydash/internal/core"
	"github.com/0xheartcode/lazydash/internal/source"
)

// Options configures which projects the GitHub source exposes. They mirror the
// config Defaults so the source owns all GitHub-specific listing policy.
type Options struct {
	Orgs           []string // additional orgs to include beyond discovered memberships
	IgnoreOrgs     []string
	IgnoreProjects []string
	OnlyOrgs       []string
	OnlyProjects   []string
}

// Source is the GitHub Projects v2 backend.
type Source struct {
	client   *api.Client
	login    string
	orgs     []string
	opts     Options
	writable bool // gh binary present, so mutations can be shelled out

	// Field-id cache from the most recently loaded board, used to translate a
	// field/option name into the ids `gh project item-edit` needs. Guarded by mu
	// because GetBoard (load) and SetField (mutate) run on different goroutines.
	mu         sync.Mutex
	lastProjID string
	fieldIDs   map[string]string            // fieldName -> fieldID
	optionIDs  map[string]map[string]string // fieldName -> optionName -> optionID
}

// New builds the GitHub source from the current gh auth context. It resolves
// the viewer login and merges discovered org memberships with configured orgs.
// It performs network calls, so it fails when gh is unauthenticated or offline.
func New(opts Options) (*Source, error) {
	client, err := api.NewClient()
	if err != nil {
		return nil, err
	}
	login, err := client.ViewerLogin()
	if err != nil {
		return nil, err
	}
	discovered, _ := client.ListViewerOrgs() // best-effort; org discovery is optional
	return &Source{
		client:   client,
		login:    login,
		orgs:     mergeOrgs(discovered, opts.Orgs),
		opts:     opts,
		writable: ghAvailable(),
	}, nil
}

// Name identifies this backend.
func (s *Source) Name() string { return "github" }

// Caps reports GitHub's capabilities: comment, close/reopen and card moves
// (single-select field edits) when the gh binary is present.
func (s *Source) Caps() source.Capabilities {
	return source.Capabilities{
		Offline:  false,
		Comment:  s.writable,
		SetState: s.writable,
		SetField: s.writable,
	}
}

// ListProjects returns the viewer's projects plus the effective orgs' projects,
// filtered by the configured include/exclude lists.
func (s *Source) ListProjects() ([]core.Project, error) {
	projects, err := s.client.ListUserProjects(s.login)
	if err != nil {
		return nil, err
	}

	// Determine the effective org list.
	var effectiveOrgs []string
	if len(s.opts.OnlyOrgs) > 0 {
		effectiveOrgs = s.opts.OnlyOrgs
	} else {
		for _, org := range s.orgs {
			if !isIgnored(org, s.opts.IgnoreOrgs) {
				effectiveOrgs = append(effectiveOrgs, org)
			}
		}
	}

	// Fetch each effective org's projects (best-effort per org).
	for _, org := range effectiveOrgs {
		orgProjects, err := s.client.ListOrgProjects(org)
		if err == nil {
			projects = append(projects, orgProjects...)
		}
	}

	// Apply the final include/exclude filter.
	if len(s.opts.OnlyProjects) > 0 {
		projects = filterAllowedProjects(projects, s.opts.OnlyProjects)
	} else {
		projects = filterIgnoredProjects(projects, s.opts.IgnoreProjects)
	}
	return projects, nil
}

// GetBoard loads a project's board from GitHub and caches its field/option ids
// so a later card move can resolve names back to ids.
func (s *Source) GetBoard(projectID string) (*core.BoardData, error) {
	bd, err := s.client.GetProjectBoard(projectID)
	if err != nil {
		return nil, err
	}
	s.cacheFieldIDs(projectID, bd)
	return bd, nil
}

// mergeOrgs unions discovered and configured orgs, case-insensitively deduped,
// preserving discovered-first order.
func mergeOrgs(discovered, configured []string) []string {
	seen := make(map[string]bool)
	var merged []string
	for _, o := range append(append([]string(nil), discovered...), configured...) {
		lo := strings.ToLower(o)
		if !seen[lo] {
			seen[lo] = true
			merged = append(merged, o)
		}
	}
	return merged
}

func isIgnored(name string, list []string) bool {
	lower := strings.ToLower(name)
	for _, item := range list {
		if strings.ToLower(item) == lower {
			return true
		}
	}
	return false
}

func filterIgnoredProjects(projects []core.Project, ignore []string) []core.Project {
	if len(ignore) == 0 {
		return projects
	}
	lower := make([]string, len(ignore))
	for i, p := range ignore {
		lower[i] = strings.ToLower(p)
	}
	var out []core.Project
	for _, p := range projects {
		ownerTitle := strings.ToLower(p.Owner + "/" + p.Title)
		bare := strings.ToLower(p.Title)
		skip := false
		for _, lp := range lower {
			if lp == ownerTitle || lp == bare {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, p)
		}
	}
	return out
}

func filterAllowedProjects(projects []core.Project, only []string) []core.Project {
	if len(only) == 0 {
		return projects
	}
	lower := make([]string, len(only))
	for i, p := range only {
		lower[i] = strings.ToLower(p)
	}
	var out []core.Project
	for _, p := range projects {
		ownerTitle := strings.ToLower(p.Owner + "/" + p.Title)
		bare := strings.ToLower(p.Title)
		for _, lp := range lower {
			if lp == ownerTitle || lp == bare {
				out = append(out, p)
				break
			}
		}
	}
	return out
}
