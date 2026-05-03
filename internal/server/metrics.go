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

// DailyCounter tracks /api/daily-reading outcomes by label.
type DailyCounter struct {
	hit   atomic.Uint64
	empty atomic.Uint64
	err   atomic.Uint64
}

func (c *DailyCounter) IncHit()        { c.hit.Add(1) }
func (c *DailyCounter) IncEmpty()      { c.empty.Add(1) }
func (c *DailyCounter) IncError()      { c.err.Add(1) }
func (c *DailyCounter) Hit() uint64    { return c.hit.Load() }
func (c *DailyCounter) Empty() uint64  { return c.empty.Load() }
func (c *DailyCounter) Errors() uint64 { return c.err.Load() }

// metricsHandler renders the Prometheus text exposition for the
// counter. v1 is counter-only; histograms are deferred.
func metricsHandler(c *ESVCallCounter, d *DailyCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprint(w, "# HELP esv_api_calls_total Successful ESV API passage fetches.\n")
		fmt.Fprint(w, "# TYPE esv_api_calls_total counter\n")
		fmt.Fprintf(w, "esv_api_calls_total %d\n", c.Value())
		fmt.Fprint(w, "# HELP daily_reading_requests_total Daily reading endpoint requests by outcome.\n")
		fmt.Fprint(w, "# TYPE daily_reading_requests_total counter\n")
		fmt.Fprintf(w, "daily_reading_requests_total{outcome=\"hit\"} %d\n", d.Hit())
		fmt.Fprintf(w, "daily_reading_requests_total{outcome=\"empty\"} %d\n", d.Empty())
		fmt.Fprintf(w, "daily_reading_requests_total{outcome=\"error\"} %d\n", d.Errors())
	}
}

// NewMetricsServer builds an http.Server bound to a localhost-only port
// (PROJECT_CONSTITUTION.md §4 — observability endpoints stay off the
// public listener).
func NewMetricsServer(addr string, c *ESVCallCounter, d *DailyCounter) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", metricsHandler(c, d))
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
