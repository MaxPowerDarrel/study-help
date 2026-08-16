package dailyreader

import (
	"bufio"
	"bytes"
	"strings"
	"time"
)

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

		readingPs, readingMsg := parsePassageCell(reading)
		if readingMsg != "" {
			msg = readingMsg
		} else {
			passages = append(passages, readingPs...)
		}

		// Psalm column: only accept passage parses. If the cell looks
		// like free text, ignore it (don't override a reading-cell
		// message or invent one).
		psalmPs, psalmMsg := parsePassageCell(psalm)
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
