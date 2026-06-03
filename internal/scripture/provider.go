// Package scripture is the translation-pluggable abstraction over
// scripture providers. Each Provider wraps one upstream API (ESV today;
// future packages add NIV, KJV, etc.). The Registry resolves a
// translation ID to the active provider; the rest of the app talks
// only to the registry.
package scripture

import (
	"context"
	"errors"
)

// ID is the canonical translation identifier. It appears verbatim in
// the ?translation= query param, the users.translation column, and the
// registry key. Uppercase, hyphenless.
type ID string

// ESV is the translation registered at launch. Additional translation
// constants land alongside their provider packages.
const ESV ID = "ESV"

// NIV is registered through internal/youversion, backed by the
// YouVersion Platform API.
const NIV ID = "NIV"

// Options bundles the renderer toggles users can flip. Each provider
// translates these into its own upstream parameters; toggles a provider
// can't honor are silently ignored. Per-provider knobs go through the
// Extra map rather than growing this struct, so adding a provider
// doesn't churn the handler signature.
//
// Extra must never be populated from arbitrary client query params —
// the handler is responsible for explicitly mapping known keys, so
// upstream behavior cannot be probed through error messages.
type Options struct {
	IncludeHeadings          bool
	IncludeFootnotes         bool
	IncludeVerseNumbers      bool
	IncludePassageReferences bool
	IncludeCrossReferences   bool
	Extra                    map[string]string
}

// Result is the rendered passage as the provider chose to emit it.
// Body is forwarded verbatim to the client; the SPA handles rendering.
type Result struct {
	Body        []byte
	ContentType string
	Status      int
}

// Provider is the abstraction every translation client implements.
// Stateless from the caller's perspective; concrete impls hold an
// HTTP client + API key.
type Provider interface {
	// ID is the registry key. Must match users.translation values and
	// the ?translation= URL param.
	ID() ID
	// DisplayName is what the SPA renders in the picker.
	DisplayName() string
	// Fetch retrieves the passage. q has already been validated against
	// the canon (see internal/canon.ValidateQuery). Implementations
	// return ErrRateLimited or ErrUpstream from this package — the
	// underlying HTTP error never leaks past the boundary.
	Fetch(ctx context.Context, q string, opts Options) (*Result, error)
}

var (
	// ErrRateLimited is returned when the upstream API rate-limited us.
	// The handler maps this to HTTP 429.
	ErrRateLimited = errors.New("scripture: rate limited")
	// ErrUpstream is returned for any other upstream failure (5xx,
	// network error, malformed response). The handler maps this to 502.
	ErrUpstream = errors.New("scripture: upstream failure")
	// ErrUnknown is returned by Registry.Get when the translation ID
	// is not registered.
	ErrUnknown = errors.New("scripture: unknown translation")
)
