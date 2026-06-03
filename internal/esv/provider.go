package esv

import (
	"context"
	"errors"

	"study-help/internal/scripture"
)

// Provider adapts Client to the scripture.Provider interface so the
// rest of the app can talk to a translation-neutral abstraction. The
// adapter is intentionally thin: it translates Options shapes and maps
// errors. Anything ESV-specific (URL, params, HTML format) stays in
// Client.
type Provider struct{ client *Client }

// NewProvider builds an ESV-backed scripture.Provider.
func NewProvider(apiKey string) *Provider {
	return &Provider{client: NewClient(apiKey)}
}

// NewProviderFromClient wraps an existing Client (typically built with
// a custom transport for tests). Production code should use NewProvider.
func NewProviderFromClient(c *Client) *Provider {
	return &Provider{client: c}
}

// ID implements scripture.Provider.
func (p *Provider) ID() scripture.ID { return scripture.ESV }

// DisplayName implements scripture.Provider.
func (p *Provider) DisplayName() string { return "English Standard Version" }

// Fetch implements scripture.Provider. ESV doesn't honor any keys from
// scripture.Options.Extra at this time; they're ignored.
func (p *Provider) Fetch(ctx context.Context, q string, o scripture.Options) (*scripture.Result, error) {
	res, err := p.client.Fetch(ctx, q, Options{
		IncludeHeadings:          o.IncludeHeadings,
		IncludeFootnotes:         o.IncludeFootnotes,
		IncludeVerseNumbers:      o.IncludeVerseNumbers,
		IncludePassageReferences: o.IncludePassageReferences,
		IncludeCrossReferences:   o.IncludeCrossReferences,
	})
	switch {
	case errors.Is(err, ErrRateLimited):
		return nil, scripture.ErrRateLimited
	case errors.Is(err, ErrUpstream):
		return nil, scripture.ErrUpstream
	case err != nil:
		return nil, err
	}
	return &scripture.Result{
		Body:        res.Body,
		ContentType: res.ContentType,
		Status:      res.Status,
	}, nil
}
