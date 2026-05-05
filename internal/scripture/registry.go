package scripture

import "sort"

// Registry resolves translation IDs to providers. Built once at startup
// (main.go) and read-only thereafter — no locks needed.
type Registry struct {
	providers map[ID]Provider
	order     []ID
	fallback  ID
}

// CatalogEntry is the SPA-facing summary of a registered provider.
type CatalogEntry struct {
	ID          ID     `json:"id"`
	DisplayName string `json:"display_name"`
}

// NewRegistry builds a registry. fallback must match one of the
// providers' IDs; it's the translation used for guests and as a
// last-resort default. Duplicate provider IDs panic — registration
// errors are programmer errors, not runtime conditions.
func NewRegistry(fallback ID, providers ...Provider) *Registry {
	r := &Registry{
		providers: make(map[ID]Provider, len(providers)),
		order:     make([]ID, 0, len(providers)),
	}
	for _, p := range providers {
		id := p.ID()
		if _, dup := r.providers[id]; dup {
			panic("scripture: duplicate provider ID " + string(id))
		}
		r.providers[id] = p
		r.order = append(r.order, id)
	}
	if _, ok := r.providers[fallback]; !ok {
		panic("scripture: fallback ID " + string(fallback) + " is not registered")
	}
	r.fallback = fallback
	return r
}

// Get returns the provider for id, or ErrUnknown if it isn't registered.
func (r *Registry) Get(id ID) (Provider, error) {
	p, ok := r.providers[id]
	if !ok {
		return nil, ErrUnknown
	}
	return p, nil
}

// Default is the registry's fallback ID — used for guests and when no
// translation is otherwise specified.
func (r *Registry) Default() ID { return r.fallback }

// Known reports whether id is a registered provider.
func (r *Registry) Known(id ID) bool {
	_, ok := r.providers[id]
	return ok
}

// Catalog returns every registered provider as a SPA-facing summary,
// sorted by ID for stable ordering across requests.
func (r *Registry) Catalog() []CatalogEntry {
	ids := make([]ID, len(r.order))
	copy(ids, r.order)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]CatalogEntry, 0, len(ids))
	for _, id := range ids {
		p := r.providers[id]
		out = append(out, CatalogEntry{ID: id, DisplayName: p.DisplayName()})
	}
	return out
}
