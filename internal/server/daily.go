package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"study-help/internal/dailyreader"
)

type dailyPassage struct {
	Book      string `json:"book"`
	Chapters  string `json:"chapters"`
	Testament string `json:"testament"`
}

type dailyPlanResult struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Passages []dailyPassage `json:"passages"`
	Message  string         `json:"message,omitempty"`
}

type dailyResponse struct {
	Plans []dailyPlanResult `json:"plans"`
}

// dailyReadingHandler returns the daily readings for the requested
// plans. ?plans=hope,bible-year selects multiple plans; missing plans
// defaults to the registry default. Outcome counters track whether at
// least one plan returned passages (hit), all plans were
// empty/special-day (empty), or the request was malformed (error).
func dailyReadingHandler(c *DailyCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tz := r.URL.Query().Get("tz")
		if tz == "" {
			c.IncError()
			http.Error(w, "missing tz", http.StatusBadRequest)
			log.Printf("daily_reading tz=\"\" outcome=error reason=missing")
			return
		}
		dateStr := r.URL.Query().Get("date")
		var now time.Time
		if dateStr == "" {
			now = time.Now()
		} else {
			// Anchor the date at midnight in the user's tz so the subsequent
			// conversion in dailyreader.Today doesn't shift it back a day for
			// users west of UTC.
			loc, err := time.LoadLocation(tz)
			if err != nil {
				c.IncError()
				http.Error(w, "invalid tz", http.StatusBadRequest)
				log.Printf("daily_reading tz=%q outcome=error reason=invalid_tz", tz)
				return
			}
			parsed, err := time.ParseInLocation("2006-01-02", dateStr, loc)
			if err != nil {
				c.IncError()
				http.Error(w, "invalid date", http.StatusBadRequest)
				log.Printf("daily_reading tz=%q outcome=error reason=invalid_date date=%q", tz, dateStr)
				return
			}
			now = parsed
		}

		planIDs := parsePlanIDs(r.URL.Query().Get("plans"))

		days, err := dailyreader.Today(planIDs, tz, now)
		if err != nil {
			c.IncError()
			switch {
			case errors.Is(err, dailyreader.ErrInvalidTZ):
				http.Error(w, "invalid tz", http.StatusBadRequest)
				log.Printf("daily_reading tz=%q outcome=error reason=invalid_tz", tz)
			case errors.Is(err, dailyreader.ErrUnknownPlan):
				http.Error(w, "invalid plan", http.StatusBadRequest)
				log.Printf("daily_reading tz=%q outcome=error reason=invalid_plan plans=%v", tz, planIDs)
			default:
				http.Error(w, "internal error", http.StatusInternalServerError)
				log.Printf("daily_reading tz=%q outcome=error err=%v", tz, err)
			}
			return
		}

		out := make([]dailyPlanResult, 0, len(days))
		anyPassages := false
		for _, d := range days {
			ps := make([]dailyPassage, 0, len(d.Passages))
			for _, p := range d.Passages {
				ps = append(ps, dailyPassage{Book: p.Book, Chapters: p.Chapters, Testament: p.Testament})
			}
			if len(ps) > 0 {
				anyPassages = true
			}
			out = append(out, dailyPlanResult{
				ID:       d.PlanID,
				Name:     d.PlanName,
				Passages: ps,
				Message:  d.Message,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dailyResponse{Plans: out})

		if anyPassages {
			c.IncHit()
			log.Printf("daily_reading tz=%q outcome=hit plans=%v", tz, planIDs)
		} else {
			c.IncEmpty()
			log.Printf("daily_reading tz=%q outcome=empty plans=%v", tz, planIDs)
		}
	}
}

// parsePlanIDs splits the comma-separated `plans` query value into IDs,
// dropping empty pieces. An empty / missing param returns nil so the
// dailyreader layer applies its default.
func parsePlanIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
