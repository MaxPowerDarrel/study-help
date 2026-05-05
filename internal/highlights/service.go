// Package highlights implements per-user range-based annotations over
// the canon. See specs/highlights.md and specs/multi-translation.md.
package highlights

import (
	"database/sql"
	"errors"
	"time"

	"study-help/internal/scripture"
)

// Service owns the highlights table and exposes HTTP handlers.
type Service struct {
	db    *sql.DB
	clock func() time.Time
	reg   *scripture.Registry
}

// Config tunes the Service. Zero-valued fields get defaults.
type Config struct {
	// Clock returns "now" for created_at. Defaults to time.Now.
	Clock func() time.Time
	// Registry resolves translation IDs. Required.
	Registry *scripture.Registry
}

// NewService constructs a Service. cfg.Registry is required.
func NewService(db *sql.DB, cfg Config) *Service {
	if cfg.Registry == nil {
		panic("highlights: Config.Registry is required")
	}
	s := &Service{db: db, clock: cfg.Clock, reg: cfg.Registry}
	if s.clock == nil {
		s.clock = time.Now
	}
	return s
}

// Highlight is one persisted range. JSON shape is the API response.
// Offsets are character positions within the textContent of the verse
// elements identified by the active translation's verse anchors.
// Ranges are half-open: [start, end). Translation scopes the row —
// highlights are only visible when the active translation matches.
type Highlight struct {
	ID          int64     `json:"id"`
	Translation string    `json:"translation"`
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
