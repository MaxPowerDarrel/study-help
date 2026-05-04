// Package highlights implements per-user range-based annotations over
// the ESV canon. See specs/highlights.md.
package highlights

import (
	"database/sql"
	"errors"
	"time"
)

// Service owns the highlights table and exposes HTTP handlers.
type Service struct {
	db    *sql.DB
	clock func() time.Time
}

// Config tunes the Service. Zero-valued fields get defaults.
type Config struct {
	// Clock returns "now" for created_at. Defaults to time.Now.
	Clock func() time.Time
}

// NewService constructs a Service.
func NewService(db *sql.DB, cfg Config) *Service {
	s := &Service{db: db, clock: cfg.Clock}
	if s.clock == nil {
		s.clock = time.Now
	}
	return s
}

// Highlight is one persisted range. JSON shape is the API response.
// Offsets are character positions within the textContent of the verse
// elements identified by ESV's include-verse-anchors. Ranges are
// half-open: [start, end).
type Highlight struct {
	ID          int64     `json:"id"`
	Book        string    `json:"book"`
	Chapter     int       `json:"chapter"`
	StartVerse  int       `json:"start_verse"`
	StartOffset int       `json:"start_offset"`
	EndVerse    int       `json:"end_verse"`
	EndOffset   int       `json:"end_offset"`
	CreatedAt   time.Time `json:"created_at"`
}

var (
	ErrNotFound = errors.New("highlights: not found")
	ErrOverlap  = errors.New("highlights: overlaps existing highlight")
)
