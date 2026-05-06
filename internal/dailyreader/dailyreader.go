package dailyreader

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"study-help/internal/canon"
	"time"
)

// ErrInvalidTZ is returned when the supplied IANA timezone name does not load.
var ErrInvalidTZ = errors.New("invalid timezone")

type Passage struct {
	Book      string
	Chapters  string
	Testament string // "OT" or "NT"
}

type Lookup struct {
	Passages []Passage
	Empty    bool // true when no row matches or both readings are empty
}

// Today returns the reading for the local date corresponding to now in tz.
// The plan is keyed by MM/DD only — any date not represented in the plan
// (including Feb 29 of leap years) yields an Empty lookup.
func Today(tz string, now time.Time) (Lookup, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return Lookup{}, ErrInvalidTZ
	}
	local := now.In(loc)
	key := local.Format("01/02")
	plan, err := parsePlan(planMarkdown)
	if err != nil {
		return Lookup{}, err
	}
	row, ok := plan[key]
	if !ok {
		return Lookup{Empty: true}, nil
	}
	var passages []Passage
	if p, ok := splitPassage(row.ot, "OT"); ok {
		passages = append(passages, p)
	}
	if p, ok := splitPassage(row.nt, "NT"); ok {
		passages = append(passages, p)
	}
	if len(passages) == 0 {
		return Lookup{Empty: true}, nil
	}
	return Lookup{Passages: passages}, nil
}

type planRow struct {
	ot, nt string
}

// parsePlan reads the embedded markdown table and returns a map keyed by MM/DD.
func parsePlan(src []byte) (map[string]planRow, error) {
	out := make(map[string]planRow, 366)
	sc := bufio.NewScanner(bytes.NewReader(src))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		// Expected: |date|ot|nt|
		fields := strings.Split(line, "|")
		// Surrounding pipes produce empty strings at indices 0 and len-1.
		if len(fields) < 5 {
			continue
		}
		date := strings.TrimSpace(fields[1])
		ot := strings.TrimSpace(fields[2])
		nt := strings.TrimSpace(fields[3])
		// Skip header and separator rows.
		if date == "" || date == "Date" || strings.HasPrefix(date, "-") {
			continue
		}
		// date is MM/DD/YY; key by MM/DD so any year wraps to the plan entry.
		if len(date) < 5 || date[2] != '/' {
			continue
		}
		key := date[:5]
		out[key] = planRow{ot: ot, nt: nt}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// splitPassage splits a cell like "Genesis 1-3" into book + chapters.
// The book name may contain spaces (e.g., "1 Samuel", "Song of Solomon");
// the chapters portion begins at the last space before a digit. The
// returned Book is normalized to its canonical canon name — the plan
// markdown uses common abbreviations ("Num.", "Matt.") that ESV happens
// to accept but the YouVersion USFM mapping rejects. canon.LookupBook
// resolves both forms; the canonical name is what every provider can
// fetch.
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
