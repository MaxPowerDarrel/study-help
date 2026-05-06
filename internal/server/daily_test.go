package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type dailyResp struct {
	Plans []dailyPlanResult `json:"plans"`
}

func decode(t *testing.T, w *httptest.ResponseRecorder) dailyResp {
	t.Helper()
	var got dailyResp
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v; body=%s", err, w.Body.String())
	}
	return got
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
	got := decode(t, w)
	if c.Hit() != 1 {
		t.Errorf("hit counter = %d, want 1", c.Hit())
	}
	if len(got.Plans) != 1 {
		t.Fatalf("got %d plans, want 1 (default plan)", len(got.Plans))
	}
	if got.Plans[0].ID != "bible-year" {
		t.Errorf("default plan id = %q, want bible-year", got.Plans[0].ID)
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

func TestDailyHandlerDateParam(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/api/daily-reading?tz=UTC&date=2026-01-01", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if len(got.Plans) != 1 || len(got.Plans[0].Passages) == 0 {
		t.Errorf("expected passages for 2026-01-01, got %+v", got.Plans)
	}
}

// Regression: prior to the fix, the date param was parsed as UTC midnight and
// then converted into the user's tz, shifting the calendar day backward for
// any tz west of UTC. With tz=America/New_York and date=2026-05-05, the
// handler previously returned May 4's reading (Numbers 17-19 / Revelation 21).
// Book names are normalized to canonical canon names; the plan markdown's
// "Num."/"Rev." are returned to the client as "Numbers"/"Revelation".
func TestDailyHandlerDateParamRespectsTZ(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet,
		"/api/daily-reading?tz=America/New_York&date=2026-05-05", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if len(got.Plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(got.Plans))
	}
	var ot, nt *dailyPassage
	for i, p := range got.Plans[0].Passages {
		switch p.Testament {
		case "OT":
			ot = &got.Plans[0].Passages[i]
		case "NT":
			nt = &got.Plans[0].Passages[i]
		}
	}
	if ot == nil || ot.Book != "Numbers" || ot.Chapters != "20-22" {
		t.Errorf("OT = %+v, want {Numbers 20-22 OT}", ot)
	}
	if nt == nil || nt.Book != "Revelation" || nt.Chapters != "22" {
		t.Errorf("NT = %+v, want {Revelation 22 NT}", nt)
	}
}

func TestDailyHandlerInvalidDateParam(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet, "/api/daily-reading?tz=UTC&date=not-a-date", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if c.Errors() != 1 {
		t.Errorf("error counter = %d, want 1", c.Errors())
	}
}

func TestDailyHandlerSpecificPlan(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet,
		"/api/daily-reading?tz=UTC&date=2026-01-12&plans=hope", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if len(got.Plans) != 1 || got.Plans[0].ID != "hope" {
		t.Fatalf("got %+v, want one hope plan", got.Plans)
	}
	// Hope plan, Jan 12: Matthew 1-4 + Psalm 1.
	hasBook := func(book string) bool {
		for _, p := range got.Plans[0].Passages {
			if p.Book == book {
				return true
			}
		}
		return false
	}
	if !hasBook("Matthew") || !hasBook("Psalms") {
		t.Errorf("hope plan 01/12 = %+v, want Matthew + Psalms", got.Plans[0].Passages)
	}
}

func TestDailyHandlerMultiplePlans(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet,
		"/api/daily-reading?tz=UTC&date=2026-01-12&plans=bible-year,hope", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if len(got.Plans) != 2 {
		t.Fatalf("got %d plans, want 2", len(got.Plans))
	}
	if got.Plans[0].ID != "bible-year" || got.Plans[1].ID != "hope" {
		t.Errorf("plan order = %s,%s; want bible-year,hope", got.Plans[0].ID, got.Plans[1].ID)
	}
}

func TestDailyHandlerSpecialDayMessage(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	// 2026-02-20 is a Hope-plan catch-up day.
	req := httptest.NewRequest(http.MethodGet,
		"/api/daily-reading?tz=UTC&date=2026-02-20&plans=hope", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if len(got.Plans) != 1 || got.Plans[0].Message == "" {
		t.Errorf("got %+v, want a single plan with a Message", got.Plans)
	}
	if c.Empty() != 1 {
		t.Errorf("empty counter = %d, want 1", c.Empty())
	}
}

func TestDailyHandlerUnknownPlan(t *testing.T) {
	c := &DailyCounter{}
	h := dailyReadingHandler(c)
	req := httptest.NewRequest(http.MethodGet,
		"/api/daily-reading?tz=UTC&plans=bogus", nil)
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
