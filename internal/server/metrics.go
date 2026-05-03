package server

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// ESVCallCounter is the in-process counter of successful ESV API calls.
// It is process-local: it resets on restart. Exposed at /metrics in
// Prometheus text exposition format.
type ESVCallCounter struct {
	n atomic.Uint64
}

func (c *ESVCallCounter) Inc()          { c.n.Add(1) }
func (c *ESVCallCounter) Value() uint64 { return c.n.Load() }

// metricsHandler renders the Prometheus text exposition for the
// counter. v1 is counter-only; histograms are deferred.
func metricsHandler(c *ESVCallCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprint(w, "# HELP esv_api_calls_total Successful ESV API passage fetches.\n")
		fmt.Fprint(w, "# TYPE esv_api_calls_total counter\n")
		fmt.Fprintf(w, "esv_api_calls_total %d\n", c.Value())
	}
}

// NewMetricsServer builds an http.Server bound to a localhost-only port
// (PROJECT_CONSTITUTION.md §4 — observability endpoints stay off the
// public listener).
func NewMetricsServer(addr string, c *ESVCallCounter) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", metricsHandler(c))
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
