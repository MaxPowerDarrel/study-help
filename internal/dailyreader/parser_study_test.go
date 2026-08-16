package dailyreader

import (
	"strconv"
	"strings"
	"study-help/internal/canon"
	"testing"
	"time"
)

// studyDates returns the plan's dates in calendar order (Jan 1 - Dec 31
// plus the Feb 29 catch-up row).
func studyDates() []string {
	var out []string
	d := time.Date(2028, time.January, 1, 0, 0, 0, 0, time.UTC) // leap year
	for d.Year() == 2028 {
		out = append(out, d.Format("01/02"))
		d = d.AddDate(0, 0, 1)
	}
	return out
}

// expand turns a Passage into its individual chapters ("Genesis 1-3" →
// Genesis 1, Genesis 2, Genesis 3).
func expand(t *testing.T, p Passage) []string {
	t.Helper()
	lo, hi, ok := strings.Cut(p.Chapters, "-")
	if !ok {
		hi = lo
	}
	first, err := strconv.Atoi(strings.TrimSpace(lo))
	if err != nil {
		t.Fatalf("bad chapters %q: %v", p.Chapters, err)
	}
	last, err := strconv.Atoi(strings.TrimSpace(hi))
	if err != nil {
		t.Fatalf("bad chapters %q: %v", p.Chapters, err)
	}
	var out []string
	for c := first; c <= last; c++ {
		out = append(out, p.Book+" "+strconv.Itoa(c))
	}
	return out
}

// studyTrack walks the plan in calendar order and returns every chapter
// carrying the given track label.
func studyTrack(t *testing.T, entries map[string]planEntry, track string) []string {
	t.Helper()
	var out []string
	for _, key := range studyDates() {
		for _, p := range entries[key].Passages {
			if p.Testament == track {
				out = append(out, expand(t, p)...)
			}
		}
	}
	return out
}

func mustParseStudy(t *testing.T) map[string]planEntry {
	t.Helper()
	entries, err := parseStudy()
	if err != nil {
		t.Fatalf("parseStudy: %v", err)
	}
	return entries
}

func TestParseStudyCoversEveryDate(t *testing.T) {
	entries := mustParseStudy(t)
	dates := studyDates()
	if len(entries) != len(dates) {
		t.Errorf("got %d rows, want %d", len(entries), len(dates))
	}
	for _, key := range dates {
		e, ok := entries[key]
		if !ok {
			t.Errorf("missing row for %s", key)
			continue
		}
		if len(e.Passages) == 0 && e.Message == "" {
			t.Errorf("row %s is empty", key)
		}
	}
}

func TestParseStudyKeyEntries(t *testing.T) {
	entries := mustParseStudy(t)

	t.Run("first day", func(t *testing.T) {
		e := entries["01/01"]
		if !hasPassage(e.Passages, "OT", "Genesis 1-2") {
			t.Errorf("missing OT Genesis 1-2 in %+v", e.Passages)
		}
		if !hasPassage(e.Passages, "NT", "Matthew 1-2") {
			t.Errorf("missing NT Matthew 1-2 in %+v", e.Passages)
		}
		if !hasPassage(e.Passages, psalmTrack, "Psalms 1") {
			t.Errorf("missing Psalm 1 in %+v", e.Passages)
		}
	})

	t.Run("book boundary splits into two passages", func(t *testing.T) {
		// 12/30 crosses Zechariah → Malachi.
		e := entries["12/30"]
		if !hasPassage(e.Passages, "OT", "Zechariah 14") ||
			!hasPassage(e.Passages, "OT", "Malachi 1") {
			t.Errorf("want Zechariah 14 + Malachi 1, got %+v", e.Passages)
		}
	})

	t.Run("leap day is a catch-up message", func(t *testing.T) {
		e := entries["02/29"]
		if e.Message == "" || len(e.Passages) != 0 {
			t.Errorf("want message-only row, got %+v", e)
		}
		if strings.Contains(e.Message, "*") {
			t.Errorf("markdown emphasis not stripped: %q", e.Message)
		}
	})

	t.Run("quarters restart the NT", func(t *testing.T) {
		for _, key := range []string{"01/01", "04/01", "07/01", "10/01"} {
			if !hasPassage(entries[key].Passages, "NT", "Matthew 1-2") {
				t.Errorf("%s: want NT restart at Matthew 1, got %+v", key, entries[key].Passages)
			}
		}
		for _, key := range []string{"03/31", "06/30", "09/30", "12/31"} {
			if !hasPassage(entries[key].Passages, "NT", "Revelation 20-22") {
				t.Errorf("%s: want NT to end at Revelation 22, got %+v", key, entries[key].Passages)
			}
		}
	})
}

// On the days Psalm 119 comes up it doubles as the Old Testament
// reading: one Psalms 119 pill, tagged OT, and no other OT chapters.
func TestParseStudyPsalm119IsTheOTReading(t *testing.T) {
	entries := mustParseStudy(t)
	for _, key := range []string{"04/29", "09/26"} {
		e := entries[key]
		var psalm119, otherOT int
		for _, p := range e.Passages {
			switch {
			case p.Book == "Psalms" && p.Chapters == "119":
				psalm119++
			case p.Testament == "OT" || p.Testament == psalmTrack:
				otherOT++
			}
		}
		if psalm119 != 1 {
			t.Errorf("%s: got %d Psalms 119 passages, want exactly 1 (%+v)", key, psalm119, e.Passages)
		}
		if otherOT != 0 {
			t.Errorf("%s: OT track should rest, got %+v", key, e.Passages)
		}
		if !hasPassage(e.Passages, "OT", "Psalms 119") {
			t.Errorf("%s: Psalms 119 should carry the OT tag, got %+v", key, e.Passages)
		}
		// The NT track keeps running on those days.
		if !hasTrack(e.Passages, "NT") {
			t.Errorf("%s: missing NT reading in %+v", key, e.Passages)
		}
	}
}

// The OT track reads Genesis → Malachi (minus Psalms, which the Psalm
// track covers) exactly once, in canonical order, across the year.
func TestParseStudyOTCoverage(t *testing.T) {
	entries := mustParseStudy(t)

	var ps119 int
	var got []string
	for _, c := range studyTrack(t, entries, "OT") {
		if c == "Psalms 119" {
			ps119++ // rides the OT track on its two days
			continue
		}
		got = append(got, c)
	}
	if ps119 != 2 {
		t.Errorf("Psalms 119 on the OT track %d time(s), want 2", ps119)
	}

	var want []string
	for _, b := range otBooksExceptPsalms() {
		for c := 1; c <= b.Chapters; c++ {
			want = append(want, b.Name+" "+strconv.Itoa(c))
		}
	}
	if len(want) != 779 {
		t.Fatalf("test setup: OT minus Psalms is %d chapters, want 779", len(want))
	}
	if len(got) != len(want) {
		t.Fatalf("OT track has %d chapters, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("OT chapter %d = %q, want %q", i+1, got[i], want[i])
		}
	}
}

// The NT track reads Matthew → Revelation four times, once per quarter.
func TestParseStudyNTCoverage(t *testing.T) {
	entries := mustParseStudy(t)
	got := studyTrack(t, entries, "NT")
	if len(got) != 4*260 {
		t.Fatalf("NT track has %d chapters, want %d", len(got), 4*260)
	}
	var pass []string
	for _, b := range ntBooks() {
		for c := 1; c <= b.Chapters; c++ {
			pass = append(pass, b.Name+" "+strconv.Itoa(c))
		}
	}
	for q := 0; q < 4; q++ {
		quarter := got[q*260 : (q+1)*260]
		for i := range pass {
			if quarter[i] != pass[i] {
				t.Fatalf("quarter %d chapter %d = %q, want %q", q+1, i+1, quarter[i], pass[i])
			}
		}
	}
}

// Every reading day carries a Psalm, and the 150-day loop covers the
// whole Psalter inside the year.
func TestParseStudyPsalmTrack(t *testing.T) {
	entries := mustParseStudy(t)
	seen := make(map[string]bool, 150)
	for _, key := range studyDates() {
		e := entries[key]
		if e.Message != "" {
			continue // leap-day catch-up row
		}
		var psalms []Passage
		for _, p := range e.Passages {
			if p.Book == "Psalms" {
				psalms = append(psalms, p)
			}
		}
		if len(psalms) != 1 {
			t.Errorf("%s: got %d Psalm passages, want 1 (%+v)", key, len(psalms), e.Passages)
			continue
		}
		seen[psalms[0].Chapters] = true
	}
	for c := 1; c <= 150; c++ {
		if !seen[strconv.Itoa(c)] {
			t.Errorf("Psalm %d never scheduled", c)
		}
	}
}

func TestDedupePassages(t *testing.T) {
	in := []Passage{
		{Book: "Psalms", Chapters: "119", Testament: "OT"},
		{Book: "Matthew", Chapters: "1-2", Testament: "NT"},
		{Book: "Psalms", Chapters: "119", Testament: psalmTrack},
	}
	got := dedupePassages(in)
	if len(got) != 2 {
		t.Fatalf("got %d passages, want 2: %+v", len(got), got)
	}
	if got[0].Testament != "OT" {
		t.Errorf("dedupe should keep the first occurrence, got %+v", got[0])
	}
}

func otBooksExceptPsalms() []canon.Book {
	var out []canon.Book
	for _, b := range canon.Canon[:39] {
		if b.Name == "Psalms" {
			continue
		}
		out = append(out, b)
	}
	return out
}

func ntBooks() []canon.Book { return canon.Canon[39:] }

func hasTrack(ps []Passage, track string) bool {
	for _, p := range ps {
		if p.Testament == track {
			return true
		}
	}
	return false
}
