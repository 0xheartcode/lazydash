package source

import "github.com/0xheartcode/lazydash/internal/core"

// Registry holds the active sources and routes work to them by name.
type Registry struct {
	sources []Source
}

// NewRegistry builds a registry over the given sources, in display order.
func NewRegistry(sources ...Source) *Registry {
	return &Registry{sources: sources}
}

// Sources returns the registered sources in order.
func (r *Registry) Sources() []Source { return r.sources }

// ByName returns the source with the given Name, or nil if none is registered.
func (r *Registry) ByName(name string) Source {
	for _, s := range r.sources {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

// ListProjects aggregates projects across every source, stamping each project
// with its owning source name. A source that errors is skipped rather than
// failing the whole listing, so one broken backend (GitHub offline) never hides
// a working one (local issues). If every source fails, the first error is
// returned so the user still sees something actionable.
func (r *Registry) ListProjects() ([]core.Project, error) {
	var all []core.Project
	var firstErr error
	for _, s := range r.sources {
		ps, err := s.ListProjects()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for i := range ps {
			ps[i].Source = s.Name()
		}
		all = append(all, ps...)
	}
	if len(all) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return all, nil
}

// GetBoard routes a board load to the source that owns the project.
func (r *Registry) GetBoard(p core.Project) (*core.BoardData, error) {
	s := r.ByName(p.Source)
	if s == nil {
		return nil, unknownSourceError(p.Source)
	}
	return s.GetBoard(p.ID)
}
