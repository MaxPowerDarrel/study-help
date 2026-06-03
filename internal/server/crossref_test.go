package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCrossrefHandlerProxiesValidRefs(t *testing.T) {
	called := atomic.Int64{}
	var gotQ, gotCrossrefs string
	stub := func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		gotQ = r.URL.Query().Get("q")
		gotCrossrefs = r.URL.Query().Get("include-crossrefs")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canonical":"Job 38:4-7","passages":["<p>xref</p>"]}`))
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := crossrefHandler(reg, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/crossref?q="+url.QueryEscape("Job 38:4-7; Psalm 33:6"), nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "<p>xref</p>") {
		t.Errorf("body missing xref content: %s", w.Body.String())
	}
	if gotQ != "Job 38:4-7; Psalm 33:6" {
		t.Errorf("upstream q = %q, want the forwarded reference list", gotQ)
	}
	// The popover must not nest cross-ref markers inside itself.
	if gotCrossrefs != "false" {
		t.Errorf("upstream include-crossrefs = %q, want false", gotCrossrefs)
	}
	if got := counter.Value(); got != 1 {
		t.Errorf("counter = %d, want 1", got)
	}
}

func TestCrossrefHandlerRejectsGarbage(t *testing.T) {
	called := atomic.Int64{}
	stub := func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := crossrefHandler(reg, counter)

	for _, q := range []string{"", "Booga 1:1", "<script>"} {
		req := httptest.NewRequest(http.MethodGet, "/api/crossref?q="+url.QueryEscape(q), nil)
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("q=%q status = %d, want 400", q, w.Code)
		}
	}
	if got := called.Load(); got != 0 {
		t.Errorf("upstream called %d times, want 0", got)
	}
	if got := counter.Value(); got != 0 {
		t.Errorf("counter = %d, want 0", got)
	}
}

func TestCrossrefHandlerSurfaces429(t *testing.T) {
	stub := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}
	reg, srv := newStubRegistry(t, stub)
	defer srv.Close()

	counter := &ESVCallCounter{}
	h := crossrefHandler(reg, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/crossref?q="+url.QueryEscape("Job 38:4-7"), nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", w.Code)
	}
	if got := counter.Value(); got != 0 {
		t.Errorf("counter = %d, want 0 on 429", got)
	}
}
