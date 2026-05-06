// Package youversion is a thin server-side HTTP client for the
// YouVersion Platform API (https://platform.youversion.com/), used to
// serve the NIV translation.
//
// The YouVersion app key never reaches the client (PROJECT_CONSTITUTION.md §4).
// All NIV passage requests go through this package.
package youversion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.youversion.com/v1/"
	// nivBibleID is YouVersion's identifier for the New International
	// Version (2011, Biblica). Verified at runtime is recommended once
	// the YouVersion app key is provisioned (GET /v1/bibles).
	nivBibleID = 111
)

// Client wraps the YouVersion Platform API.
type Client struct {
	appKey  string
	baseURL string
	bibleID int
	http    *http.Client
}

// NewClient builds a Client backed by the given app key.
func NewClient(appKey string) *Client {
	return &Client{
		appKey:  appKey,
		baseURL: defaultBaseURL,
		bibleID: nivBibleID,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Options selects formatting toggles. Today YouVersion only honors
// IncludeHeadings (maps to ?include_headings=true, which adds
// <div class="s1 yv-h">Title</div> section headings). The other
// toggles are kept on the struct so the scripture.Provider adapter
// can pass scripture.Options through with the same shape as the ESV
// adapter; they're currently no-ops upstream.
type Options struct {
	IncludeHeadings          bool
	IncludeFootnotes         bool
	IncludeVerseNumbers      bool
	IncludePassageReferences bool
}

// Result is what the SPA consumes — JSON-shaped to match the existing
// ESV envelope so web/src/api.ts's decoder works for both translations
// without branching.
type Result struct {
	Body        []byte
	ContentType string
	Status      int
}

// ErrRateLimited is returned when YouVersion responds with HTTP 429.
var ErrRateLimited = errors.New("youversion rate limited")

// ErrUpstream is returned for non-429 YouVersion failures (timeout,
// 5xx, malformed body, etc.).
var ErrUpstream = errors.New("youversion upstream failure")

// passageJSON is the YouVersion Platform passage response shape from
// /v1/bibles/{id}/passages/{usfm}?format=html.
type passageJSON struct {
	Reference string `json:"reference"`
	Content   string `json:"content"`
}

// envelopeJSON mirrors the ESV `/passage/html` envelope so both
// providers hand the SPA the same shape.
type envelopeJSON struct {
	Canonical string   `json:"canonical"`
	Passages  []string `json:"passages"`
}

// Fetch retrieves the rendered NIV passage. q is the canonical user
// reference (e.g. "John 3:1-21"). canon.ValidateQuery has already
// rejected obvious garbage, so toUSFM only has to handle the four
// allow-listed shapes.
func (c *Client) Fetch(ctx context.Context, q string, opts Options) (*Result, error) {
	usfm, err := toUSFM(q)
	if err != nil {
		return nil, fmt.Errorf("translate q to USFM: %w", err)
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	u.Path = path.Join(u.Path, "bibles", strconv.Itoa(c.bibleID), "passages", usfm)
	// format=html switches the response from plain text to USX-rendered
	// HTML with verse-boundary anchors (<span class="yv-v" v="N">),
	// verse labels (<span class="yv-vlbl">N</span>), and red-letter
	// markup (<span class="wj">). The default is text-only.
	// include_headings=true adds <div class="s1 yv-h">Title</div>
	// section headings; omitted when the user has turned them off.
	q2 := u.Query()
	q2.Set("format", "html")
	if opts.IncludeHeadings {
		q2.Set("include_headings", "true")
	}
	u.RawQuery = q2.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-YVP-App-Key", c.appKey)
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

	var pr passageJSON
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("%w: decode body: %v", ErrUpstream, err)
	}

	canonical := pr.Reference
	if canonical == "" {
		canonical = q
	}
	// Match ESV's wire format: emit raw <,> rather than <,> so
	// the bytes the SPA receives look the same regardless of provider.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(envelopeJSON{
		Canonical: canonical,
		Passages:  []string{pr.Content},
	}); err != nil {
		return nil, fmt.Errorf("%w: marshal envelope: %v", ErrUpstream, err)
	}

	return &Result{
		Body:        bytes.TrimRight(buf.Bytes(), "\n"),
		ContentType: "application/json",
		Status:      resp.StatusCode,
	}, nil
}

// toUSFM translates a canon-validated reference (e.g. "John 3:1-21")
// into a USFM passage reference suitable for the YouVersion endpoint
// (e.g. "JHN.3.1-JHN.3.21"). Handles the four shapes accepted by
// canon.ValidateQuery.
func toUSFM(q string) (string, error) {
	q = strings.TrimSpace(q)

	// Match canon.ValidateQuery's split rule: tail starts at the last
	// space-before-digit.
	tailStart := -1
	for i := len(q) - 1; i > 0; i-- {
		if q[i] == ' ' && i+1 < len(q) && q[i+1] >= '0' && q[i+1] <= '9' {
			tailStart = i
			break
		}
	}
	if tailStart <= 0 {
		return "", fmt.Errorf("missing chapter in %q", q)
	}
	bookName := strings.TrimSpace(q[:tailStart])
	tail := strings.TrimSpace(q[tailStart+1:])
	code := usfmBook(bookName)
	if code == "" {
		return "", fmt.Errorf("unknown book %q", bookName)
	}

	chapterPart, versePart, hasColon := strings.Cut(tail, ":")
	if !hasColon {
		startCh, endCh, hasDash := cutDash(chapterPart)
		if hasDash {
			return fmt.Sprintf("%s.%s-%s.%s", code, startCh, code, endCh), nil
		}
		return fmt.Sprintf("%s.%s", code, chapterPart), nil
	}
	startV, endV, hasDash := cutDash(versePart)
	if hasDash {
		return fmt.Sprintf("%s.%s.%s-%s.%s.%s",
			code, chapterPart, startV, code, chapterPart, endV), nil
	}
	return fmt.Sprintf("%s.%s.%s", code, chapterPart, versePart), nil
}

func cutDash(s string) (string, string, bool) {
	a, b, ok := strings.Cut(s, "-")
	return strings.TrimSpace(a), strings.TrimSpace(b), ok
}
