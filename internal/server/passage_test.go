package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"study-help/internal/esv"
	"study-help/internal/scripture"
)

// newStubRegistry builds a scripture.Registry whose ESV provider is
// backed by a stubbed esv.Client pointing at the given test handler.
// The stub host rewriter swaps the upstream URL transparently so the
// real Client's request shape (params, headers, error mapping) is
// exercised end-to-end.
func newStubRegistry(t *testing.T, handler http.HandlerFunc) (*scripture.Registry, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := esv.NewClient("test-token")
	rewriter := &hostRewriter{
		target: srv.URL,
		next:   http.DefaultTransport,
	}
	httpClient := esv.HTTPClientForTest(c)
	httpClient.Transport = rewriter
	httpClient.Timeout = 5 * time.Second
	reg := scripture.NewRegistry(scripture.ESV, esv.NewProviderFromClient(c))
	return reg, srv
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

func TestPassageHandlerRejectsMalformedQ(t *testing.T) {
	called := atomic.Int64{}
	stub := func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := passageHandler(reg, counter)

	for _, q := range []string{"!!!", "Booga 99", ""} {
		req := httptest.NewRequest(http.MethodGet, "/api/passage?q="+url.QueryEscape(q), nil)
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("q=%q got status %d, want 400", q, w.Code)
		}
	}
	if got := called.Load(); got != 0 {
		t.Errorf("upstream called %d times, want 0", got)
	}
	if got := counter.Value(); got != 0 {
		t.Errorf("counter = %d, want 0", got)
	}
}

func TestPassageHandlerProxiesAndCounts(t *testing.T) {
	called := atomic.Int64{}
	stub := func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"passages":["<p>...</p>"]}`))
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := passageHandler(reg, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/passage?q="+url.QueryEscape("John 3:1-21"), nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "passages") {
		t.Errorf("body missing passages: %q", body)
	}
	if got := counter.Value(); got != 1 {
		t.Errorf("counter = %d, want 1", got)
	}
	if got := called.Load(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestPassageHandlerSurfaces429(t *testing.T) {
	stub := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := passageHandler(reg, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/passage?q=John+3", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}
	if got := counter.Value(); got != 0 {
		t.Errorf("counter = %d, want 0 on 429", got)
	}
}

func TestPassageHandlerSurfacesUpstream5xx(t *testing.T) {
	stub := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := passageHandler(reg, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/passage?q=John+3", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", w.Code)
	}
}

func TestPassageHandlerRejectsUnknownTranslation(t *testing.T) {
	stub := func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called when translation is invalid")
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := passageHandler(reg, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/passage?q=John+3&translation=BOGUS", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestMetricsExposesCounter(t *testing.T) {
	counter := &ESVCallCounter{}
	counter.Inc()
	counter.Inc()
	counter.Inc()

	srv := NewMetricsServer("127.0.0.1:0", counter, &DailyCounter{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	srv.Handler.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "# TYPE esv_api_calls_total counter") {
		t.Errorf("missing TYPE line: %s", body)
	}
	if !strings.Contains(body, "esv_api_calls_total 3") {
		t.Errorf("missing sample line: %s", body)
	}
}

var _ = context.Background
