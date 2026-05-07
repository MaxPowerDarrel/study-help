package youversion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestToUSFM(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"John 3", "JHN.3"},
		{"John 3:16", "JHN.3.16"},
		{"John 3:1-21", "JHN.3.1-JHN.3.21"},
		{"Psalms 119-120", "PSA.119-PSA.120"},
		// Hope reading-plan cells use the singular "Psalm"; canon.LookupBook
		// resolves it to "Psalms" so the USFM lookup still finds PSA.
		{"Psalm 5", "PSA.5"},
		{"Psalm 119:33-40", "PSA.119.33-PSA.119.40"},
		{"1 Corinthians 13:4-7", "1CO.13.4-1CO.13.7"},
		{"Song of Solomon 2:10", "SNG.2.10"},
		{"Song of Songs 2:10", "SNG.2.10"},
		{"3 John 1", "3JN.1"},
	}
	for _, c := range cases {
		got, err := toUSFM(c.in)
		if err != nil {
			t.Errorf("toUSFM(%q) err = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("toUSFM(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToUSFM_UnknownBook(t *testing.T) {
	if _, err := toUSFM("Gibberish 3"); err == nil {
		t.Error("expected error for unknown book, got nil")
	}
}

func TestExpandChapterRange(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		// Single chapter — pass through.
		{"Genesis 1", []string{"Genesis 1"}},
		// Chapter range — fan out per-chapter so each call hits the
		// YouVersion endpoint with one passage at a time.
		{"Genesis 1-3", []string{"Genesis 1", "Genesis 2", "Genesis 3"}},
		{"Matthew 11-12", []string{"Matthew 11", "Matthew 12"}},
		// Verse-range queries pass through (YouVersion handles them).
		{"Psalm 119:33-40", []string{"Psalm 119:33-40"}},
		{"John 3:16", []string{"John 3:16"}},
		// Multi-word book names still parse correctly.
		{"Song of Solomon 1-2", []string{"Song of Solomon 1", "Song of Solomon 2"}},
	}
	for _, c := range cases {
		got := expandChapterRange(c.in)
		if len(got) != len(c.want) {
			t.Errorf("expandChapterRange(%q) = %v (len %d), want %v (len %d)",
				c.in, got, len(got), c.want, len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("expandChapterRange(%q)[%d] = %q, want %q",
					c.in, i, got[i], c.want[i])
			}
		}
	}
}

// withStub builds a Client whose transport is rewritten to point at
// the provided test server. Mirrors the pattern in
// internal/server/passage_test.go.
func withStub(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("test-key")
	httpClient := HTTPClientForTest(c)
	httpClient.Transport = &hostRewriter{target: srv.URL, next: http.DefaultTransport}
	return c
}

type hostRewriter struct {
	target string
	next   http.RoundTripper
}

func (h *hostRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	u, err := url.Parse(h.target)
	if err != nil {
		return nil, err
	}
	r2 := req.Clone(req.Context())
	r2.URL.Scheme = u.Scheme
	r2.URL.Host = u.Host
	r2.Host = u.Host
	return h.next.RoundTrip(r2)
}

func TestFetch_HappyPath_RewrapsEnvelope(t *testing.T) {
	var gotPath, gotHeader, gotFormat, gotHeadings string
	stub := func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("X-YVP-App-Key")
		gotFormat = r.URL.Query().Get("format")
		gotHeadings = r.URL.Query().Get("include_headings")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","bible_id":111,"reference":"John 3:16","content":"<p>For God so loved the world.</p>"}`))
	}
	c := withStub(t, stub)
	res, err := c.Fetch(context.Background(), "John 3:16", Options{IncludeHeadings: true})
	if err != nil {
		t.Fatalf("Fetch err = %v", err)
	}
	if !strings.HasSuffix(gotPath, "/v1/bibles/111/passages/JHN.3.16") {
		t.Errorf("unexpected upstream path: %s", gotPath)
	}
	if gotHeader != "test-key" {
		t.Errorf("missing X-YVP-App-Key header, got %q", gotHeader)
	}
	if gotFormat != "html" {
		t.Errorf("missing format=html on request, got %q", gotFormat)
	}
	if gotHeadings != "true" {
		t.Errorf("expected include_headings=true on request, got %q", gotHeadings)
	}
	if res.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want application/json", res.ContentType)
	}
	var env envelopeJSON
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("envelope decode: %v", err)
	}
	if env.Canonical != "John 3:16" {
		t.Errorf("Canonical = %q, want John 3:16", env.Canonical)
	}
	if len(env.Passages) != 1 || !strings.Contains(env.Passages[0], "For God so loved") {
		t.Errorf("Passages = %v", env.Passages)
	}
}

func TestFetch_OmitsIncludeHeadingsWhenFalse(t *testing.T) {
	var gotHeadings, gotPresent string
	stub := func(w http.ResponseWriter, r *http.Request) {
		gotHeadings = r.URL.Query().Get("include_headings")
		if r.URL.Query().Has("include_headings") {
			gotPresent = "yes"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","reference":"John 3","content":"<p></p>"}`))
	}
	c := withStub(t, stub)
	if _, err := c.Fetch(context.Background(), "John 3", Options{IncludeHeadings: false}); err != nil {
		t.Fatalf("Fetch err = %v", err)
	}
	if gotPresent == "yes" || gotHeadings != "" {
		t.Errorf("include_headings should not be sent when disabled, got %q (present=%q)", gotHeadings, gotPresent)
	}
}

func TestFetch_429_ReturnsErrRateLimited(t *testing.T) {
	c := withStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, err := c.Fetch(context.Background(), "John 3", Options{})
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

func TestFetch_5xx_ReturnsErrUpstream(t *testing.T) {
	c := withStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, err := c.Fetch(context.Background(), "John 3", Options{})
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("err = %v, want ErrUpstream", err)
	}
}

func TestFetch_MalformedJSON_ReturnsErrUpstream(t *testing.T) {
	c := withStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	})
	_, err := c.Fetch(context.Background(), "John 3", Options{})
	if !errors.Is(err, ErrUpstream) {
		t.Errorf("err = %v, want ErrUpstream", err)
	}
}
