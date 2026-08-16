package dailyreader

import (
	"strconv"
	"strings"
	"study-help/internal/canon"
)

// bookTestament tags every canonical book with its testament. Used by
// the plan parsers whose cells mix OT and NT books.
var bookTestament = map[string]string{
	"Genesis": "OT", "Exodus": "OT", "Leviticus": "OT", "Numbers": "OT", "Deuteronomy": "OT",
	"Joshua": "OT", "Judges": "OT", "Ruth": "OT",
	"1 Samuel": "OT", "2 Samuel": "OT", "1 Kings": "OT", "2 Kings": "OT",
	"1 Chronicles": "OT", "2 Chronicles": "OT", "Ezra": "OT", "Nehemiah": "OT", "Esther": "OT",
	"Job": "OT", "Psalms": "OT", "Proverbs": "OT", "Ecclesiastes": "OT", "Song of Solomon": "OT",
	"Isaiah": "OT", "Jeremiah": "OT", "Lamentations": "OT", "Ezekiel": "OT", "Daniel": "OT",
	"Hosea": "OT", "Joel": "OT", "Amos": "OT", "Obadiah": "OT", "Jonah": "OT", "Micah": "OT",
	"Nahum": "OT", "Habakkuk": "OT", "Zephaniah": "OT", "Haggai": "OT", "Zechariah": "OT", "Malachi": "OT",
	"Matthew": "NT", "Mark": "NT", "Luke": "NT", "John": "NT", "Acts": "NT",
	"Romans": "NT", "1 Corinthians": "NT", "2 Corinthians": "NT", "Galatians": "NT",
	"Ephesians": "NT", "Philippians": "NT", "Colossians": "NT",
	"1 Thessalonians": "NT", "2 Thessalonians": "NT", "1 Timothy": "NT", "2 Timothy": "NT",
	"Titus": "NT", "Philemon": "NT", "Hebrews": "NT", "James": "NT",
	"1 Peter": "NT", "2 Peter": "NT", "1 John": "NT", "2 John": "NT", "3 John": "NT",
	"Jude": "NT", "Revelation": "NT",
}

// parsePassageCell parses one table cell into either a list of passages
// (when the cell looks like a passage list) or a free-text message (when
// it doesn't, e.g. "Catch-up day!", "**Happy Thanksgiving!**", or a
// quote). Segments are separated by "&" or ";" — the Hope plan uses
// both, the study plan uses ";" for book-boundary crossings. The
// returned passages all have Testament populated via bookTestament; the
// canon resolver supplies the canonical book name.
func parsePassageCell(cell string) ([]Passage, string) {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return nil, ""
	}
	parts := strings.FieldsFunc(cell, func(r rune) bool {
		return r == '&' || r == ';'
	})
	var ps []Passage
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		p, ok := parsePassageRef(part)
		if !ok {
			return nil, cleanMessage(cell)
		}
		ps = append(ps, p)
	}
	if len(ps) == 0 {
		return nil, cleanMessage(cell)
	}
	return ps, ""
}

// parsePassageRef handles a single passage segment from a plan cell.
// Accepts both "Book N" / "Book N-M" / "Book N:V-W" forms and book-only
// forms ("Philippians", "Obadiah") which are expanded to the book's full
// chapter range via canon.LookupBook.
func parsePassageRef(s string) (Passage, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Passage{}, false
	}
	// Normalise en-dash to hyphen so downstream parsers (and the SPA's
	// chapter-range expander) see a consistent separator.
	s = strings.ReplaceAll(s, "–", "-")

	if p, ok := splitPassage(s, ""); ok {
		if _, known := canon.LookupBook(p.Book); !known {
			return Passage{}, false
		}
		// Drop verse ranges ("119:33-40" → "119"); the daily panel
		// renders chapter blocks. The verse-range fidelity is a known
		// v1 limitation documented in specs/multi-plan.md.
		if i := strings.Index(p.Chapters, ":"); i >= 0 {
			p.Chapters = p.Chapters[:i]
		}
		p.Testament = bookTestament[p.Book]
		return p, true
	}

	// Book-only form, e.g. "Philippians" → "Philippians 1-4".
	if b, ok := canon.LookupBook(s); ok {
		chapters := "1"
		if b.Chapters > 1 {
			chapters = "1-" + strconv.Itoa(b.Chapters)
		}
		return Passage{
			Book:      b.Name,
			Chapters:  chapters,
			Testament: bookTestament[b.Name],
		}, true
	}
	return Passage{}, false
}

func cleanMessage(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "**", ""))
}
