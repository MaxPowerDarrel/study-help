package dailyreader

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"study-help/internal/canon"
	"time"
)

// bookTestament tags every canonical book with its testament. Used by
// the Hope plan parser, whose "Reading" column mixes OT and NT books.
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

// parseHope parses the embedded Hope-plan markdown into a map keyed by
// MM/DD. Format differs from the Bible-in-One-Year table:
//   - Header is "| Date | Reading | Psalm |".
//   - Dates are "Jan 12" form (no year).
//   - Reading cells may contain multiple passages joined by "&" or ";"
//     (e.g. "1 Kings 1-5 & 1 Chronicles 1-9" or "Titus; Philemon").
//   - Cells may be book-only references for short books ("Philippians",
//     "Obadiah") — expanded to the full book chapter range here.
//   - Psalm cells may include verse ranges ("Psalm 18:1–15"); the verse
//     range is dropped (we render the full chapter, since the daily
//     panel is chapter-block based).
//   - Special rows ("Catch-up day!", bolded holidays, scripture quotes)
//     yield a planEntry with Message set instead of Passages.
func parseHope() (map[string]planEntry, error) {
	out := make(map[string]planEntry, 260)
	sc := bufio.NewScanner(bytes.NewReader(hopeMarkdown))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 5 {
			continue
		}
		date := strings.TrimSpace(fields[1])
		reading := strings.TrimSpace(fields[2])
		psalm := strings.TrimSpace(fields[3])
		if date == "" || date == "Date" || strings.HasPrefix(date, "-") {
			continue
		}
		t, err := time.Parse("Jan 2", date)
		if err != nil {
			continue
		}
		key := t.Format("01/02")

		var passages []Passage
		var msg string

		readingPs, readingMsg := parseHopeCell(reading)
		if readingMsg != "" {
			msg = readingMsg
		} else {
			passages = append(passages, readingPs...)
		}

		// Psalm column: only accept passage parses. If the cell looks
		// like free text, ignore it (don't override a reading-cell
		// message or invent one).
		psalmPs, psalmMsg := parseHopeCell(psalm)
		if psalmMsg == "" {
			passages = append(passages, psalmPs...)
		}

		out[key] = planEntry{Passages: passages, Message: msg}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// parseHopeCell parses a Hope-plan cell into either a list of passages
// (when the cell looks like a passage list) or a free-text message
// (when it doesn't, e.g. "Catch-up day!", "**Happy Thanksgiving!**", or
// a quote). The returned passages all have Testament populated via
// bookTestament; the canon resolver supplies the canonical book name.
func parseHopeCell(cell string) ([]Passage, string) {
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
		p, ok := parseHopePassage(part)
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

// parseHopePassage handles a single passage segment from a Hope-plan
// cell. Accepts both "Book N" / "Book N-M" / "Book N:V-W" forms and
// book-only forms ("Philippians", "Obadiah") which are expanded to the
// book's full chapter range via canon.LookupBook.
func parseHopePassage(s string) (Passage, bool) {
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
