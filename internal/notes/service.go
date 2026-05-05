// Package notes implements per-user range-anchored prose annotations
// over the canon. See specs/notes.md and specs/multi-translation.md.
package notes

import (
	"database/sql"
	"errors"
	"time"

	"study-help/internal/scripture"
)

// MaxBodyBytes caps a single note's body. Server rejects POST/PATCH
// bodies larger than this with HTTP 400 (specs/notes.md, 2026-05-04).
const MaxBodyBytes = 16 * 1024

// Service owns the notes table and exposes HTTP handlers.
type Service struct {
	db    *sql.DB
	clock func() time.Time
	reg   *scripture.Registry
}

// Config tunes the Service. Zero-valued fields get defaults.
type Config struct {
	// Clock returns "now" for created_at / updated_at. Defaults to time.Now.
	Clock func() time.Time
	// Registry resolves translation IDs. Required.
	Registry *scripture.Registry
}

// NewService constructs a Service. cfg.Registry is required.
func NewService(db *sql.DB, cfg Config) *Service {
	if cfg.Registry == nil {
		panic("notes: Config.Registry is required")
	}
	s := &Service{db: db, clock: cfg.Clock, reg: cfg.Registry}
	if s.clock == nil {
		s.clock = time.Now
	}
	return s
}

// Note is one persisted range-anchored note. Half-open ranges, like
// highlights — but unlike highlights, multiple notes per range are
// allowed (no overlap rejection on create). Translation scopes the
// row — notes are only visible when the active translation matches.
type Note struct {
	ID          int64     `json:"id"`
	Translation string    `json:"translation"`
	Book        string    `json:"book"`
	Chapter     int       `json:"chapter"`
	StartVerse  int       `json:"start_verse"`
	StartOffset int       `json:"start_offset"`
	EndVerse    int       `json:"end_verse"`
	EndOffset   int       `json:"end_offset"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var ErrNotFound = errors.New("notes: not found")
