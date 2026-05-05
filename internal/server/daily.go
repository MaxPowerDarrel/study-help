package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"study-help/internal/dailyreader"
)

type dailyPassage struct {
	Book      string `json:"book"`
	Chapters  string `json:"chapters"`
	Testament string `json:"testament"`
}

type dailyResponse struct {
	Passages []dailyPassage `json:"passages"`
	Message  string         `json:"message,omitempty"`
}

// dailyReadingHandler returns today's OT/NT readings for the client's
// timezone. 200 with empty passages means "no reading"; invalid tz is 400.
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
		lookup, err := dailyreader.Today(tz, now)
		if err != nil {
			c.IncError()
			if errors.Is(err, dailyreader.ErrInvalidTZ) {
				http.Error(w, "invalid tz", http.StatusBadRequest)
				log.Printf("daily_reading tz=%q outcome=error reason=invalid_tz", tz)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("daily_reading tz=%q outcome=error err=%v", tz, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if lookup.Empty {
			c.IncEmpty()
			_ = json.NewEncoder(w).Encode(dailyResponse{
				Passages: []dailyPassage{},
				Message:  "No reading for today",
			})
			log.Printf("daily_reading tz=%q outcome=empty", tz)
			return
		}

		c.IncHit()
		out := make([]dailyPassage, 0, len(lookup.Passages))
		for _, p := range lookup.Passages {
			out = append(out, dailyPassage{
				Book:      p.Book,
				Chapters:  p.Chapters,
				Testament: p.Testament,
			})
		}
		_ = json.NewEncoder(w).Encode(dailyResponse{Passages: out})
		log.Printf("daily_reading tz=%q outcome=hit n=%d", tz, len(out))
	}
}
