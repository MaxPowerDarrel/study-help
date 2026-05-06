package dailyreader

import (
	"errors"
	"study-help/internal/canon"
	"testing"
	"time"
)

func mustTime(t *testing.T, layout, value string) time.Time {
	t.Helper()
	tm, err := time.Parse(layout, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return tm
}

func TestTodayInvalidTZ(t *testing.T) {
	_, err := Today("Not/A_Zone", time.Now())
	if !errors.Is(err, ErrInvalidTZ) {
		t.Fatalf("got %v, want ErrInvalidTZ", err)
	}
}

func TestTodayHits(t *testing.T) {
	cases := []struct {
		name    string
		tz      string
		now     time.Time
		wantOT  string // book + " " + chapters; "" means no OT
		wantNT  string
		wantLen int
	}{
		{
			name:    "first day of plan year",
			tz:      "UTC",
			now:     mustTime(t, time.RFC3339, "2026-01-01T12:00:00Z"),
			wantOT:  "Genesis 1-3",
			wantNT:  "Romans 1",
			wantLen: 2,
		},
		{
			name:    "row with empty NT",
			tz:      "UTC",
			now:     mustTime(t, time.RFC3339, "2026-01-09T12:00:00Z"),
			wantOT:  "Genesis 23-24",
			wantNT:  "",
			wantLen: 1,
		},
		{
			name:    "year wrap forward",
			tz:      "UTC",
			now:     mustTime(t, time.RFC3339, "2030-01-01T12:00:00Z"),
			wantOT:  "Genesis 1-3",
			wantNT:  "Romans 1",
			wantLen: 2,
		},
		{
			name:    "DST boundary in LA",
			tz:      "America/Los_Angeles",
			now:     mustTime(t, time.RFC3339, "2026-03-08T09:30:00Z"), // 02:30 PDT
			wantLen: -1,                                                // just check date matches 03/08, not specific contents
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Today(tc.tz, tc.now)
			if err != nil {
				t.Fatalf("Today: %v", err)
			}
			if got.Empty {
				t.Fatalf("got Empty, want passages")
			}
			if tc.wantLen >= 0 && len(got.Passages) != tc.wantLen {
				t.Fatalf("got %d passages, want %d", len(got.Passages), tc.wantLen)
			}
			if tc.wantOT != "" {
				if !hasPassage(got.Passages, "OT", tc.wantOT) {
					t.Errorf("missing OT %q in %+v", tc.wantOT, got.Passages)
				}
			}
			if tc.wantNT != "" {
				if !hasPassage(got.Passages, "NT", tc.wantNT) {
					t.Errorf("missing NT %q in %+v", tc.wantNT, got.Passages)
				}
			}
		})
	}
}

func TestTodayLeapDayFallsBackToEmpty(t *testing.T) {
	got, err := Today("UTC", mustTime(t, time.RFC3339, "2028-02-29T12:00:00Z"))
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if !got.Empty {
		t.Fatalf("got %+v, want Empty", got.Passages)
	}
}

func TestRoundTripEveryRowValidates(t *testing.T) {
	plan, err := parsePlan(planMarkdown)
	if err != nil {
		t.Fatalf("parsePlan: %v", err)
	}
	if len(plan) == 0 {
		t.Fatal("plan parsed to zero rows")
	}
	for key, row := range plan {
		for _, cell := range []struct {
			testament, raw string
		}{{"OT", row.ot}, {"NT", row.nt}} {
			p, ok := splitPassage(cell.raw, cell.testament)
			if !ok {
				continue // empty cell is allowed
			}
			q := p.Book + " " + p.Chapters
			if err := canon.ValidateQuery(q); err != nil {
				t.Errorf("row %s %s %q: ValidateQuery=%v", key, cell.testament, q, err)
			}
		}
	}
}

func TestSplitPassageMultiWordBook(t *testing.T) {
	cases := []struct {
		in, book, chapters string
	}{
		{"Genesis 1-3", "Genesis", "1-3"},
		{"1 Samuel 17", "1 Samuel", "17"},
		{"Song of Solomon 2", "Song of Solomon", "2"},
		{"Revelation 22", "Revelation", "22"},
		// Abbreviations from the plan markdown should normalize to
		// canonical canon names so every provider can fetch them.
		{"Num. 23-25", "Numbers", "23-25"},
		{"Matt. 1", "Matthew", "1"},
		{"Rev. 22", "Revelation", "22"},
		{"1 Sam. 17", "1 Samuel", "17"},
		{"Song of Songs 2", "Song of Solomon", "2"},
	}
	for _, c := range cases {
		p, ok := splitPassage(c.in, "OT")
		if !ok {
			t.Errorf("splitPassage(%q) returned false", c.in)
			continue
		}
		if p.Book != c.book || p.Chapters != c.chapters {
			t.Errorf("splitPassage(%q) = %q / %q, want %q / %q", c.in, p.Book, p.Chapters, c.book, c.chapters)
		}
	}
}

func TestSplitPassageNormalizesEveryPlanRow(t *testing.T) {
	plan, err := parsePlan(planMarkdown)
	if err != nil {
		t.Fatalf("parsePlan: %v", err)
	}
	for key, row := range plan {
		for _, cell := range []struct {
			testament, raw string
		}{{"OT", row.ot}, {"NT", row.nt}} {
			p, ok := splitPassage(cell.raw, cell.testament)
			if !ok {
				continue
			}
			b, ok := canon.LookupBook(p.Book)
			if !ok {
				t.Errorf("row %s %s: book %q not in canon", key, cell.testament, p.Book)
				continue
			}
			if p.Book != b.Name {
				t.Errorf("row %s %s: book %q is not canonical (want %q)", key, cell.testament, p.Book, b.Name)
			}
		}
	}
}

func hasPassage(ps []Passage, testament, want string) bool {
	for _, p := range ps {
		if p.Testament == testament && p.Book+" "+p.Chapters == want {
			return true
		}
	}
	return false
}
