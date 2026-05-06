package dailyreader

import (
	"bufio"
	"bytes"
	"strings"
)

// parseBibleYear parses the embedded "Bible in One Year" markdown table
// into a map keyed by MM/DD. Each entry has up to two passages (OT, NT).
// Format: |Date (MM/DD/YY)|OT Reading|NT Reading|
func parseBibleYear() (map[string]planEntry, error) {
	out := make(map[string]planEntry, 366)
	sc := bufio.NewScanner(bytes.NewReader(bibleYearMarkdown))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		fields := strings.Split(line, "|")
		// Surrounding pipes produce empty strings at indices 0 and len-1.
		if len(fields) < 5 {
			continue
		}
		date := strings.TrimSpace(fields[1])
		ot := strings.TrimSpace(fields[2])
		nt := strings.TrimSpace(fields[3])
		if date == "" || date == "Date" || strings.HasPrefix(date, "-") {
			continue
		}
		// date is MM/DD/YY; key by MM/DD so any year wraps to the plan entry.
		if len(date) < 5 || date[2] != '/' {
			continue
		}
		key := date[:5]
		var passages []Passage
		if p, ok := splitPassage(ot, "OT"); ok {
			passages = append(passages, p)
		}
		if p, ok := splitPassage(nt, "NT"); ok {
			passages = append(passages, p)
		}
		out[key] = planEntry{Passages: passages}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
