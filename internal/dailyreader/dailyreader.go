package dailyreader

import (
	"errors"
	"strings"
	"study-help/internal/canon"
	"sync"
	"time"
)

// ErrInvalidTZ is returned when the supplied IANA timezone name does not load.
var ErrInvalidTZ = errors.New("invalid timezone")

// ErrUnknownPlan is returned when a requested plan ID is not registered.
var ErrUnknownPlan = errors.New("unknown plan")

// DefaultPlanID is used when Today is called with no plan IDs.
const DefaultPlanID = "bible-year"

// Passage is one OT/NT/Psalm slot inside a daily reading.
type Passage struct {
	Book      string
	Chapters  string
	Testament string // track label: "OT", "NT", or "Psalm"
}

// Day is the lookup result for one plan on one date.
type Day struct {
	PlanID   string
	PlanName string
	Passages []Passage
	Message  string // populated for special / no-reading days
}

// Today returns one Day per requested plan ID, in request order.
// Empty planIDs falls back to [DefaultPlanID]. An unknown ID returns
// ErrUnknownPlan; an invalid tz returns ErrInvalidTZ.
func Today(planIDs []string, tz string, now time.Time) ([]Day, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, ErrInvalidTZ
	}
	if len(planIDs) == 0 {
		planIDs = []string{DefaultPlanID}
	}
	resolved := make([]*plan, 0, len(planIDs))
	for _, id := range planIDs {
		p := findPlan(id)
		if p == nil {
			return nil, ErrUnknownPlan
		}
		resolved = append(resolved, p)
	}
	key := now.In(loc).Format("01/02")
	out := make([]Day, 0, len(resolved))
	for _, p := range resolved {
		entries, err := p.get()
		if err != nil {
			return nil, err
		}
		d := Day{PlanID: p.id, PlanName: p.name}
		entry, ok := entries[key]
		switch {
		case !ok:
			d.Message = "No reading for today"
		case entry.Message != "":
			d.Message = entry.Message
		case len(entry.Passages) == 0:
			d.Message = "No reading for today"
		default:
			d.Passages = entry.Passages
		}
		out = append(out, d)
	}
	return out, nil
}

// planEntry is one row in a parsed plan: either a list of passages or
// a free-text message (e.g. "Catch-up day!").
type planEntry struct {
	Passages []Passage
	Message  string
}

type plan struct {
	id, name string
	parse    func() (map[string]planEntry, error)

	once  sync.Once
	cache map[string]planEntry
	err   error
}

func (p *plan) get() (map[string]planEntry, error) {
	p.once.Do(func() { p.cache, p.err = p.parse() })
	return p.cache, p.err
}

// plans is the package-level registry. Order is the order in which
// plans are listed in the catalog and rendered in the SPA when
// multiple plans are active.
var plans = []*plan{
	{id: "bible-year", name: "Bible in One Year", parse: parseBibleYear},
	{id: "hope", name: "Hope (2026)", parse: parseHope},
	{id: "study", name: "Study Plan", parse: parseStudy},
}

func findPlan(id string) *plan {
	for _, p := range plans {
		if p.id == id {
			return p
		}
	}
	return nil
}

// splitPassage splits a cell like "Genesis 1-3" into book + chapters.
// The book name may contain spaces (e.g., "1 Samuel", "Song of Solomon");
// the chapters portion begins at the last space before a digit. The
// returned Book is normalized to its canonical canon name when found —
// the plan markdowns use common abbreviations ("Num.", "Matt.") that
// ESV happens to accept but the YouVersion USFM mapping rejects.
// canon.LookupBook resolves both forms; the canonical name is what
// every provider can fetch.
func splitPassage(cell, testament string) (Passage, bool) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return Passage{}, false
	}
	cut := -1
	for i := len(cell) - 1; i > 0; i-- {
		if cell[i] == ' ' && i+1 < len(cell) && cell[i+1] >= '0' && cell[i+1] <= '9' {
			cut = i
			break
		}
	}
	if cut <= 0 {
		return Passage{}, false
	}
	rawBook := strings.TrimSpace(cell[:cut])
	bookName := rawBook
	if b, ok := canon.LookupBook(rawBook); ok {
		bookName = b.Name
	}
	return Passage{
		Book:      bookName,
		Chapters:  strings.TrimSpace(cell[cut+1:]),
		Testament: testament,
	}, true
}
