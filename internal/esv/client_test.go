package esv

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// captureQuery runs a Fetch with the given options against a stub that
// records the upstream request's query params, then returns them.
func captureQuery(t *testing.T, opts Options) url.Values {
	t.Helper()
	var got url.Values
	c := NewClient("test-key")
	HTTPClientForTest(c).Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		got = r.URL.Query()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"passages":["x"]}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	if _, err := c.Fetch(context.Background(), "John 1", opts); err != nil {
		t.Fatalf("Fetch err = %v", err)
	}
	return got
}

func TestClientEmitsCrossrefParams(t *testing.T) {
	on := captureQuery(t, Options{IncludeCrossReferences: true})
	if got := on.Get("include-crossrefs"); got != "true" {
		t.Errorf("include-crossrefs = %q, want true", got)
	}
	// crossref-url is always sent (empty) so the marker href is the bare
	// reference string the SPA parses.
	if !on.Has("crossref-url") {
		t.Errorf("crossref-url param missing; query = %v", on)
	}
	if got := on.Get("crossref-url"); got != "" {
		t.Errorf("crossref-url = %q, want empty", got)
	}

	off := captureQuery(t, Options{IncludeCrossReferences: false})
	if got := off.Get("include-crossrefs"); got != "false" {
		t.Errorf("include-crossrefs = %q, want false", got)
	}
}
