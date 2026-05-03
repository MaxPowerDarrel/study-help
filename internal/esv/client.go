// Package esv is a thin server-side HTTP client for api.esv.org.
//
// The ESV API key never reaches the client (PROJECT_CONSTITUTION.md §4).
// All scripture requests go through this package.
package esv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.esv.org/v3/passage/html/"

// Client wraps the ESV API.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewClient builds a Client backed by the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Options selects which formatting elements ESV should include in its
// rendered HTML. Each toggle maps to a query param on api.esv.org.
type Options struct {
	IncludeHeadings          bool
	IncludeFootnotes         bool
	IncludeVerseNumbers      bool
	IncludePassageReferences bool
	IncludeWordOfChrist      bool
}

// Result is a verbatim ESV API response.
type Result struct {
	Body        []byte
	ContentType string
	Status      int
}

// ErrRateLimited is returned when ESV responds with HTTP 429.
var ErrRateLimited = errors.New("esv rate limited")

// ErrUpstream is returned for non-429 ESV failures (timeout, 5xx, etc.).
var ErrUpstream = errors.New("esv upstream failure")

// Fetch retrieves the rendered passage from ESV. q is the user reference
// (e.g. "John 3:1-21"). The caller is expected to have already validated
// q against the allow-list.
func (c *Client) Fetch(ctx context.Context, q string, opts Options) (*Result, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	params := url.Values{}
	params.Set("q", q)
	params.Set("include-headings", boolParam(opts.IncludeHeadings))
	params.Set("include-footnotes", boolParam(opts.IncludeFootnotes))
	params.Set("include-verse-numbers", boolParam(opts.IncludeVerseNumbers))
	params.Set("include-passage-references", boolParam(opts.IncludePassageReferences))
	params.Set("include-word-of-christ", boolParam(opts.IncludeWordOfChrist))
	// Audio playback is a Non-goal in specs/passage-reader.md; ESV's
	// passage/html defaults this to true and would otherwise emit a
	// <a class="mp3link">Listen</a> link in every payload.
	params.Set("include-audio-link", "false")
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstream, err)
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	return &Result{
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
		Status:      resp.StatusCode,
	}, nil
}

func boolParam(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
