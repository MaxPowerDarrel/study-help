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
// rejected obvious garbage upstream of this call.
//
// Chapter-range queries ("Genesis 1-3") are fanned out into per-chapter
// requests because the YouVersion endpoint accepts a single passage per
// call, not a chapter range. Verse-range queries ("Psalm 119:33-40")
// and single chapters pass through as one request.
func (c *Client) Fetch(ctx context.Context, q string, opts Options) (*Result, error) {
	pieces := expandChapterRange(q)

	passages := make([]string, 0, len(pieces))
	var canonical string
	for _, piece := range pieces {
		pr, err := c.fetchOne(ctx, piece, opts)
		if err != nil {
			return nil, err
		}
		if canonical == "" {
			canonical = pr.Reference
		}
		passages = append(passages, pr.Content)
	}
	if canonical == "" {
		canonical = q
	}
	// For multi-chapter ranges, prefer the original user-facing q over
	// the first chapter's "Genesis 1" canonical.
	if len(pieces) > 1 {
		canonical = q
	}

	// Match ESV's wire format: emit raw <,> rather than &lt;,&gt; so
	// the bytes the SPA receives look the same regardless of provider.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(envelopeJSON{
		Canonical: canonical,
		Passages:  passages,
	}); err != nil {
		return nil, fmt.Errorf("%w: marshal envelope: %v", ErrUpstream, err)
	}

	return &Result{
		Body:        bytes.TrimRight(buf.Bytes(), "\n"),
		ContentType: "application/json",
		Status:      http.StatusOK,
	}, nil
}

// fetchOne issues a single GET against the passages endpoint. q must
// be a single chapter or a verse-range within a single chapter — see
// expandChapterRange.
func (c *Client) fetchOne(ctx context.Context, q string, opts Options) (*passageJSON, error) {
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
	return &pr, nil
}

// expandChapterRange splits a chapter-range query ("Genesis 1-3") into
// per-chapter queries (["Genesis 1", "Genesis 2", "Genesis 3"]). Single
// chapters and verse-range queries pass through unchanged. The
// YouVersion passage endpoint only accepts one passage per call, so
// this fan-out happens at the provider boundary; the SPA still issues
// one request per pill.
func expandChapterRange(q string) []string {
	q = strings.TrimSpace(q)

	// Mirror canon.ValidateQuery's split rule.
	tailStart := -1
	for i := len(q) - 1; i > 0; i-- {
		if q[i] == ' ' && i+1 < len(q) && q[i+1] >= '0' && q[i+1] <= '9' {
			tailStart = i
			break
		}
	}
	if tailStart <= 0 {
		return []string{q}
	}
	book := strings.TrimSpace(q[:tailStart])
	tail := strings.TrimSpace(q[tailStart+1:])

	a, b, ok := strings.Cut(tail, "-")
	if !ok {
		return []string{q}
	}
	start, err1 := strconv.Atoi(strings.TrimSpace(a))
	end, err2 := strconv.Atoi(strings.TrimSpace(b))
	if err1 != nil || err2 != nil || start < 1 || end < start {
		return []string{q}
	}
	out := make([]string, 0, end-start+1)
	for c := start; c <= end; c++ {
		out = append(out, fmt.Sprintf("%s %d", book, c))
	}
	return out
}

// toUSFM translates a canon-validated reference into a USFM passage
// reference suitable for the YouVersion endpoint. Handles the two
// shapes accepted by canon.ValidateQuery: "<book> <chapter>" and
// "<book> <chapter>-<chapter>". Verse-level references were retired
// on 2026-05-07 (NIV/ESV parity).
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

	startCh, endCh, hasDash := strings.Cut(tail, "-")
	if hasDash {
		return fmt.Sprintf("%s.%s-%s.%s",
			code, strings.TrimSpace(startCh),
			code, strings.TrimSpace(endCh)), nil
	}
	return fmt.Sprintf("%s.%s", code, tail), nil
}
