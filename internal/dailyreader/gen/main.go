// Command gen writes internal/dailyreader/study-plan.md, the generated
// markdown table behind the "Study Plan" reading plan.
//
// Plan shape (see specs/study-plan.md):
//
//   - OT track: the whole Old Testament except Psalms (779 chapters),
//     spread evenly over the year in canonical order.
//   - NT track: the whole New Testament (260 chapters), restarted at the
//     top of every calendar quarter — four complete passes per year.
//   - Psalm track: Psalms 1-150 on a straight 150-day loop, so the
//     Psalter is covered inside the year and then some.
//   - When the day's Psalm is 119, Psalm 119 *is* that day's OT reading:
//     the OT track pauses and the day's chapters go to the next day.
//
// The table is keyed MM/DD (year-agnostic, like the other plans) and
// runs Jan 1 - Dec 31 on a 365-day calendar. Feb 29 is a catch-up day.
//
// Run with: go generate ./internal/dailyreader/...
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"study-help/internal/canon"
)

// refYear is a non-leap year used only for date arithmetic; the emitted
// table is keyed MM/DD and carries no year.
const refYear = 2026

// psalmCycle is the length of the daily Psalm loop.
const psalmCycle = 150

// longPsalm rests the OT track: on the day it comes up it serves as both
// the Psalm and the Old Testament reading.
const longPsalm = 119

// chapterRef is one chapter of one book.
type chapterRef struct {
	book    string
	chapter int
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run() error {
	days := daysOfYear()

	// Psalm for each day: straight 1..150 loop from Jan 1.
	psalms := make([]int, len(days))
	for i := range days {
		psalms[i] = i%psalmCycle + 1
	}

	// OT track skips the days Psalm 119 takes over.
	otDays := make([]int, 0, len(days))
	for i := range days {
		if psalms[i] != longPsalm {
			otDays = append(otDays, i)
		}
	}
	otCells := make([]string, len(days))
	for j, chunk := range spread(otChapters(), len(otDays)) {
		otCells[otDays[j]] = formatCell(chunk)
	}

	// NT track: a full pass per calendar quarter.
	ntCells := make([]string, len(days))
	for _, q := range quarters(days) {
		for j, chunk := range spread(ntChapters(), len(q)) {
			ntCells[q[j]] = formatCell(chunk)
		}
	}

	var b strings.Builder
	b.WriteString(header())
	for i, d := range days {
		ot, psalm := otCells[i], fmt.Sprintf("Psalm %d", psalms[i])
		if psalms[i] == longPsalm {
			ot = psalm
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", d.Format("01/02"), ot, ntCells[i], psalm)
		if d.Month() == time.February && d.Day() == 28 {
			b.WriteString("| 02/29 | **Catch-up day — no new reading** | | |\n")
		}
	}
	return os.WriteFile("study-plan.md", []byte(b.String()), 0o644)
}

// daysOfYear returns every day of the reference (non-leap) year.
func daysOfYear() []time.Time {
	var out []time.Time
	d := time.Date(refYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	for d.Year() == refYear {
		out = append(out, d)
		d = d.AddDate(0, 0, 1)
	}
	return out
}

// quarters groups day indices into the four calendar quarters.
func quarters(days []time.Time) [][]int {
	out := make([][]int, 4)
	for i, d := range days {
		q := (int(d.Month()) - 1) / 3
		out[q] = append(out[q], i)
	}
	return out
}

func otChapters() []chapterRef { return chaptersIn(canon.Canon[:39], "Psalms") }

func ntChapters() []chapterRef { return chaptersIn(canon.Canon[39:]) }

// chaptersIn flattens books into a chapter list, skipping any excluded book.
func chaptersIn(books []canon.Book, exclude ...string) []chapterRef {
	skip := make(map[string]bool, len(exclude))
	for _, name := range exclude {
		skip[name] = true
	}
	var out []chapterRef
	for _, b := range books {
		if skip[b.Name] {
			continue
		}
		for c := 1; c <= b.Chapters; c++ {
			out = append(out, chapterRef{b.Name, c})
		}
	}
	return out
}

// spread divides chapters into n contiguous chunks, sized as evenly as
// the division allows (each chunk gets floor or ceil of the average,
// with the longer chunks distributed across the whole span).
func spread(chapters []chapterRef, n int) [][]chapterRef {
	out := make([][]chapterRef, n)
	total := len(chapters)
	for j := 0; j < n; j++ {
		start := total * j / n
		end := total * (j + 1) / n
		out[j] = chapters[start:end]
	}
	return out
}

// formatCell renders a run of chapters as a table cell: consecutive chapters
// of one book collapse to "Book 1-3", and a run that crosses a book
// boundary emits both refs joined by "; " ("Genesis 50; Exodus 1").
func formatCell(chunk []chapterRef) string {
	var parts []string
	for i := 0; i < len(chunk); {
		j := i
		for j+1 < len(chunk) &&
			chunk[j+1].book == chunk[i].book &&
			chunk[j+1].chapter == chunk[j].chapter+1 {
			j++
		}
		if i == j {
			parts = append(parts, fmt.Sprintf("%s %d", chunk[i].book, chunk[i].chapter))
		} else {
			parts = append(parts, fmt.Sprintf("%s %d-%d", chunk[i].book, chunk[i].chapter, chunk[j].chapter))
		}
		i = j + 1
	}
	return strings.Join(parts, "; ")
}

func header() string {
	return `<!-- Generated by internal/dailyreader/gen. Do not edit by hand; run: go generate ./internal/dailyreader/... -->

# Study Plan

A year through the Old Testament, the New Testament once a quarter, and a
Psalm every day.

- **OT** — the whole Old Testament except Psalms, in canonical order,
  spread across the year (about 2-3 chapters a day).
- **NT** — the whole New Testament, restarted every calendar quarter, so
  it is read four times in the year (about 3 chapters a day).
- **Psalm** — Psalms 1-150 on a repeating 150-day loop from Jan 1.
- **Psalm 119** — on the days it comes up (Apr 29 and Sep 26) it is also
  that day's Old Testament reading; the OT track rests.

Dates are keyed MM/DD, so the plan repeats every year. Feb 29 is a
catch-up day.

## Reading Schedule

| Date | OT Reading | NT Reading | Psalm |
|------|------------|------------|-------|
`
}
