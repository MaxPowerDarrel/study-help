package dailyreader

import (
	"testing"
)

func TestParseHopeKeyEntries(t *testing.T) {
	plan, err := parseHope()
	if err != nil {
		t.Fatalf("parseHope: %v", err)
	}

	cases := []struct {
		key    string
		desc   string
		check  func(t *testing.T, e planEntry)
		exists bool
	}{
		{
			key:    "01/12",
			desc:   "first plan day: Matthew 1-4 + Psalm 1",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				if e.Message != "" {
					t.Errorf("unexpected Message %q", e.Message)
				}
				if !hasPassage(e.Passages, "NT", "Matthew 1-4") {
					t.Errorf("missing Matthew 1-4 in %+v", e.Passages)
				}
				if !hasPassage(e.Passages, "OT", "Psalms 1") {
					t.Errorf("missing Psalms 1 in %+v", e.Passages)
				}
			},
		},
		{
			key:    "02/04",
			desc:   "verse range Psalm gets stripped to chapter",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				// Feb 4 — Psalm 18:1–15 → Psalms 18 (verse range dropped).
				if !hasPassage(e.Passages, "OT", "Psalms 18") {
					t.Errorf("Psalms 18 not present in %+v", e.Passages)
				}
				if hasPassage(e.Passages, "OT", "Psalms 18:1-15") {
					t.Errorf("verse range should be stripped, found %+v", e.Passages)
				}
			},
		},
		{
			key:    "02/20",
			desc:   "catch-up day yields a message",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				if e.Message == "" {
					t.Errorf("expected Message, got %+v", e)
				}
			},
		},
		{
			key:    "04/03",
			desc:   "Good Friday quote yields a message",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				if e.Message == "" {
					t.Errorf("expected Message, got %+v", e)
				}
			},
		},
		{
			key:    "07/02",
			desc:   "multi-passage row with & separator",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				if !hasPassage(e.Passages, "OT", "1 Kings 1-5") {
					t.Errorf("missing 1 Kings 1-5 in %+v", e.Passages)
				}
				if !hasPassage(e.Passages, "OT", "1 Chronicles 1-9") {
					t.Errorf("missing 1 Chronicles 1-9 in %+v", e.Passages)
				}
			},
		},
		{
			key:    "08/14",
			desc:   "multi-book row with ; separator and book-only refs",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				// Titus has 3 chapters; Philemon has 1.
				if !hasPassage(e.Passages, "NT", "Titus 1-3") {
					t.Errorf("missing Titus 1-3 in %+v", e.Passages)
				}
				if !hasPassage(e.Passages, "NT", "Philemon 1") {
					t.Errorf("missing Philemon 1 in %+v", e.Passages)
				}
			},
		},
		{
			key:    "11/26",
			desc:   "Thanksgiving holiday yields a stripped message",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				if e.Message == "" {
					t.Errorf("expected Message, got %+v", e)
				}
				if e.Message == "" || containsAny(e.Message, []string{"**"}) {
					t.Errorf("Message %q still has ** markers", e.Message)
				}
			},
		},
		{
			key:    "12/25",
			desc:   "Christmas quote yields a message",
			exists: true,
			check: func(t *testing.T, e planEntry) {
				if e.Message == "" {
					t.Errorf("expected Message, got %+v", e)
				}
			},
		},
		{
			key:    "02/29",
			desc:   "leap day not in Hope plan",
			exists: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.key+"_"+tc.desc, func(t *testing.T) {
			entry, ok := plan[tc.key]
			if ok != tc.exists {
				t.Fatalf("plan[%q] present=%v, want %v", tc.key, ok, tc.exists)
			}
			if !ok {
				return
			}
			tc.check(t, entry)
		})
	}
}

func TestParseHopeCellMessageFallback(t *testing.T) {
	cases := []struct {
		in  string
		msg string
	}{
		{"Catch-up day!", "Catch-up day!"},
		{"**Happy Thanksgiving!**", "Happy Thanksgiving!"},
		{"**Merry Christmas!**", "Merry Christmas!"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			ps, msg := parsePassageCell(tc.in)
			if len(ps) != 0 {
				t.Errorf("got passages %+v, want none", ps)
			}
			if msg != tc.msg {
				t.Errorf("got msg %q, want %q", msg, tc.msg)
			}
		})
	}
}

func TestParseHopePassageBookOnly(t *testing.T) {
	cases := []struct {
		in    string
		book  string
		chaps string
	}{
		{"Philippians", "Philippians", "1-4"},
		{"Obadiah", "Obadiah", "1"},
		{"Ruth", "Ruth", "1-4"},
		{"James", "James", "1-5"},
	}
	for _, c := range cases {
		p, ok := parsePassageRef(c.in)
		if !ok {
			t.Errorf("parsePassageRef(%q) returned false", c.in)
			continue
		}
		if p.Book != c.book || p.Chapters != c.chaps {
			t.Errorf("parsePassageRef(%q) = %q / %q, want %q / %q", c.in, p.Book, p.Chapters, c.book, c.chaps)
		}
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
