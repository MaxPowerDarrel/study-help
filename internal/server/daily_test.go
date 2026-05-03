package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type dailyResp struct {
	Passages []dailyPassage `json:"passages"`
	Message  string         `json:"message,omitempty"`
}

func TestDailyHandlerHit(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/api/daily-reading?tz=UTC", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var got dailyResp
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v; body=%s", err, w.Body.String())
	}
	if c.Hit() != 1 {
		t.Errorf("hit counter = %d, want 1", c.Hit())
	}
	// We don't pin the date (depends on time.Now), but hit means non-empty.
	if got.Message != "" {
		t.Errorf("hit response carried message %q", got.Message)
	}
	if len(got.Passages) == 0 {
		t.Errorf("hit response had no passages")
	}
}

func TestDailyHandlerInvalidTZ(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/api/daily-reading?tz=Not/Real", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if c.Errors() != 1 {
		t.Errorf("error counter = %d, want 1", c.Errors())
	}
}

func TestDailyHandlerMissingTZ(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/api/daily-reading", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if c.Errors() != 1 {
		t.Errorf("error counter = %d, want 1", c.Errors())
	}
}

func TestMetricsExposesDailyCounter(t *testing.T) {
	esvCounter := &ESVCallCounter{}
	dailyCounter := &DailyCounter{}
	dailyCounter.IncHit()
	dailyCounter.IncEmpty()
	dailyCounter.IncEmpty()
	dailyCounter.IncError()

	srv := NewMetricsServer("127.0.0.1:0", esvCounter, dailyCounter)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	srv.Handler.ServeHTTP(w, r)

	body := w.Body.String()
	for _, want := range []string{
		`# TYPE daily_reading_requests_total counter`,
		`daily_reading_requests_total{outcome="hit"} 1`,
		`daily_reading_requests_total{outcome="empty"} 2`,
		`daily_reading_requests_total{outcome="error"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q\nbody:\n%s", want, body)
		}
	}
}
