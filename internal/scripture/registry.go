package scripture

// Registry resolves translation IDs to providers. Built once at startup
// (main.go) and read-only thereafter — no locks needed.
type Registry struct {
	providers map[ID]Provider
	order     []ID
	fallback  ID
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
