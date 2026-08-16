package dailyreader

import (
	"bufio"
	"bytes"
	"strings"
)

// psalmTrack is the track label carried by passages from the study
// plan's Psalm column, so the daily panel can tag that pill "Psalm"
// instead of a second "OT".
const psalmTrack = "Psalm"

// parseStudy parses the embedded study-plan markdown into a map keyed
// by MM/DD. Format: "| MM/DD | OT Reading | NT Reading | Psalm |".
//
//   - Dates carry no year — the plan repeats annually. Feb 29 is a row
//     of its own (a catch-up day).
//   - OT and NT cells may cross a book boundary, in which case the refs
//     are joined by ";" ("Genesis 50; Exodus 1") and yield one Passage
//     each.
//   - Psalm-column passages are tagged psalmTrack rather than "OT".
//   - On the days the Psalm is 119 the OT cell repeats it (Psalm 119 is
//     that day's Old Testament reading); the duplicate is collapsed to a
//     single passage, which keeps the OT-column tag.
//   - A cell that doesn't parse as passages yields a Message instead
//     (the Feb 29 catch-up row).
//
// The table is generated — see internal/dailyreader/gen.
func parseStudy() (map[string]planEntry, error) {
	out := make(map[string]planEntry, 366)
	sc := bufio.NewScanner(bytes.NewReader(studyMarkdown))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "|") {
			continue
		}
		fields := strings.Split(line, "|")
		// Surrounding pipes produce empty strings at index 0 and len-1.
		if len(fields) < 6 {
			continue
		}
		key := strings.TrimSpace(fields[1])
		if len(key) != 5 || key[2] != '/' {
			continue // header row, separator row, or malformed date
		}

		var passages []Passage
		var msg string

		otPs, otMsg := parsePassageCell(fields[2])
		if otMsg != "" {
			msg = otMsg
		} else {
			passages = append(passages, otPs...)
		}

		ntPs, ntMsg := parsePassageCell(fields[3])
		if ntMsg == "" {
			passages = append(passages, ntPs...)
		}

		// Psalm column is conservative in the same way the Hope parser
		// is: free text there never invents or overrides a message.
		psalmPs, psalmMsg := parsePassageCell(fields[4])
		if psalmMsg == "" {
			for _, p := range psalmPs {
				p.Testament = psalmTrack
				passages = append(passages, p)
			}
		}

		out[key] = planEntry{Passages: dedupePassages(passages), Message: msg}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// dedupePassages drops repeats of the same book + chapter range within a
// single day, keeping the first occurrence (and so its track label).
func dedupePassages(ps []Passage) []Passage {
	if len(ps) < 2 {
		return ps
	}
	seen := make(map[string]bool, len(ps))
	out := ps[:0:0]
	for _, p := range ps {
		k := p.Book + " " + p.Chapters
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, p)
	}
	return out
}
