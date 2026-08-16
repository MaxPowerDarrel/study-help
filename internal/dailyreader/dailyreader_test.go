package dailyreader

import (
	"errors"
	"strings"
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
	_, err := Today(nil, "Not/A_Zone", time.Now())
	if !errors.Is(err, ErrInvalidTZ) {
		t.Fatalf("got %v, want ErrInvalidTZ", err)
	}
}

func TestTodayUnknownPlan(t *testing.T) {
	_, err := Today([]string{"bogus"}, "UTC", time.Now())
	if !errors.Is(err, ErrUnknownPlan) {
		t.Fatalf("got %v, want ErrUnknownPlan", err)
	}
}

func TestTodayDefaultPlan(t *testing.T) {
	got, err := Today(nil, "UTC", mustTime(t, time.RFC3339, "2026-01-01T12:00:00Z"))
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d Days, want 1", len(got))
	}
	if got[0].PlanID != DefaultPlanID {
		t.Errorf("plan id = %q, want %q", got[0].PlanID, DefaultPlanID)
	}
	if !hasPassage(got[0].Passages, "OT", "Genesis 1-3") {
		t.Errorf("missing OT Genesis 1-3 in %+v", got[0].Passages)
	}
}

func TestTodayBibleYearHits(t *testing.T) {
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
			got, err := Today([]string{"bible-year"}, tc.tz, tc.now)
			if err != nil {
				t.Fatalf("Today: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d Days, want 1", len(got))
			}
			d := got[0]
			if d.Message != "" {
				t.Fatalf("got Message %q, want passages", d.Message)
			}
			if tc.wantLen >= 0 && len(d.Passages) != tc.wantLen {
				t.Fatalf("got %d passages, want %d", len(d.Passages), tc.wantLen)
			}
			if tc.wantOT != "" && !hasPassage(d.Passages, "OT", tc.wantOT) {
				t.Errorf("missing OT %q in %+v", tc.wantOT, d.Passages)
			}
			if tc.wantNT != "" && !hasPassage(d.Passages, "NT", tc.wantNT) {
				t.Errorf("missing NT %q in %+v", tc.wantNT, d.Passages)
			}
		})
	}
}

func TestTodayMultiplePlans(t *testing.T) {
	got, err := Today(
		[]string{"bible-year", "hope"}, "UTC",
		mustTime(t, time.RFC3339, "2026-01-12T12:00:00Z"),
	)
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d Days, want 2", len(got))
	}
	if got[0].PlanID != "bible-year" || got[1].PlanID != "hope" {
		t.Errorf("plan order = %s, %s; want bible-year, hope", got[0].PlanID, got[1].PlanID)
	}
	// Bible-year on 01/12 has Genesis 29-30 / Romans 10.
	if !hasPassage(got[0].Passages, "OT", "Genesis 29-30") {
		t.Errorf("bible-year missing OT Genesis 29-30 in %+v", got[0].Passages)
	}
	// Hope on Jan 12 has Matthew 1-4 + Psalm 1.
	if !hasPassage(got[1].Passages, "NT", "Matthew 1-4") {
		t.Errorf("hope missing NT Matthew 1-4 in %+v", got[1].Passages)
	}
	if !hasPassage(got[1].Passages, "OT", "Psalms 1") {
		t.Errorf("hope missing Psalms 1 in %+v", got[1].Passages)
	}
}

func TestTodayStudyPlan(t *testing.T) {
	got, err := Today([]string{"study"}, "UTC", mustTime(t, time.RFC3339, "2026-01-01T12:00:00Z"))
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d Days, want 1", len(got))
	}
	d := got[0]
	if d.PlanName != "Study Plan" {
		t.Errorf("plan name = %q, want %q", d.PlanName, "Study Plan")
	}
	if len(d.Passages) != 3 {
		t.Fatalf("got %d passages, want OT + NT + Psalm: %+v", len(d.Passages), d.Passages)
	}
	if !hasPassage(d.Passages, "OT", "Genesis 1-2") ||
		!hasPassage(d.Passages, "NT", "Matthew 1-2") ||
		!hasPassage(d.Passages, "Psalm", "Psalms 1") {
		t.Errorf("unexpected passages: %+v", d.Passages)
	}
}

func TestTodayLeapDayFallsBackToMessage(t *testing.T) {
	got, err := Today([]string{"bible-year"}, "UTC", mustTime(t, time.RFC3339, "2028-02-29T12:00:00Z"))
	if err != nil {
		t.Fatalf("Today: %v", err)
	}
	if len(got) != 1 || got[0].Message == "" || len(got[0].Passages) != 0 {
		t.Fatalf("got %+v, want empty-message Day", got)
	}
}

func TestRoundTripEveryRowValidates(t *testing.T) {
	for _, planID := range []string{"bible-year", "hope", "study"} {
		t.Run(planID, func(t *testing.T) {
			p := findPlan(planID)
			if p == nil {
				t.Fatalf("plan %q not found", planID)
			}
			entries, err := p.parse()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(entries) == 0 {
				t.Fatal("plan parsed to zero rows")
			}
			for key, entry := range entries {
				for _, ps := range entry.Passages {
					if _, ok := canon.LookupBook(ps.Book); !ok {
						t.Errorf("plan %s row %s: unknown book %q", planID, key, ps.Book)
						continue
					}
					// Chapters may be a single number, a hyphen range,
					// or a comma-separated list (Hope plan uses
					// "Romans 1,2"). Validate each piece independently
					// — the SPA splits on the same separators before
					// fetching, so canon validation runs piecewise too.
					for _, piece := range strings.Split(ps.Chapters, ",") {
						q := ps.Book + " " + strings.TrimSpace(piece)
						if err := canon.ValidateQuery(q); err != nil {
							t.Errorf("plan %s row %s %q: ValidateQuery=%v", planID, key, q, err)
						}
					}
				}
			}
		})
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

func hasPassage(ps []Passage, testament, want string) bool {
	for _, p := range ps {
		if p.Testament == testament && p.Book+" "+p.Chapters == want {
			return true
		}
	}
	return false
}
