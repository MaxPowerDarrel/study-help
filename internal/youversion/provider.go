package youversion

import (
	"context"
	"errors"

	"study-help/internal/scripture"
)

// Provider adapts Client to the scripture.Provider interface so the
// rest of the app can talk to a translation-neutral abstraction. The
// adapter is intentionally thin: it translates Options shapes and maps
// errors. Anything YouVersion-specific (URL, headers, USFM, response
// envelope rewriting) stays in Client.
type Provider struct{ client *Client }

// NewProvider builds a YouVersion-backed scripture.Provider for NIV.
func NewProvider(appKey string) *Provider {
	return &Provider{client: NewClient(appKey)}
}

// NewProviderFromClient wraps an existing Client (typically built with
// a custom transport for tests). Production code should use NewProvider.
func NewProviderFromClient(c *Client) *Provider {
	return &Provider{client: c}
}

// ID implements scripture.Provider.
func (p *Provider) ID() scripture.ID { return scripture.NIV }

// DisplayName implements scripture.Provider.
func (p *Provider) DisplayName() string { return "New International Version" }

// Fetch implements scripture.Provider. YouVersion's passage endpoint
// renders a fixed HTML shape and doesn't honor per-request formatting
// toggles; the keys are forwarded for future use without behavioral
// change today. In particular it has no cross-reference parameter, so
// scripture.Options.IncludeCrossReferences is silently ignored — NIV
// passages carry no .cf markers and never trigger the cross-ref popover.
func (p *Provider) Fetch(ctx context.Context, q string, o scripture.Options) (*scripture.Result, error) {
	res, err := p.client.Fetch(ctx, q, Options{
		IncludeHeadings:          o.IncludeHeadings,
		IncludeFootnotes:         o.IncludeFootnotes,
		IncludeVerseNumbers:      o.IncludeVerseNumbers,
		IncludePassageReferences: o.IncludePassageReferences,
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
