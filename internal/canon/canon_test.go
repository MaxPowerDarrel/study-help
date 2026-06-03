package canon

import (
	"strings"
	"testing"
)

func TestValidateQueryAccepts(t *testing.T) {
	cases := []string{
		"John 3",
		"Genesis 1",
		"Revelation 22",
		"Psalms 23",
		"Psalm 23",
		"3 John 1",
		"Malachi 4",
		"Obadiah 1",
		"Genesis 1-3",
		"Numbers 15-16",
		"Revelation 21-22",
		"1 Samuel 1-2",
	}
	for _, q := range cases {
		if err := ValidateQuery(q); err != nil {
			t.Errorf("ValidateQuery(%q) = %v, want nil", q, err)
		}
	}
}

func TestValidateQueryRejects(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"!!!",
		"Booga 99",
		"John",
		"John 99",
		"John 0",
		// Verse-level references were retired 2026-05-07 (NIV/ESV
		// parity); previously accepted, now rejected.
		"John 3:16",
		"John 3:1-21",
		"1 Samuel 17:45-50",
		"Song of Solomon 2:1",
		"<script>",
		"John 3; DROP TABLE",
		"Genesis 0-3",
		"Genesis 3-1",
		"Genesis 1-51",
		"Genesis 1-",
		"Genesis -3",
	}
	for _, q := range cases {
		if err := ValidateQuery(q); err == nil {
			t.Errorf("ValidateQuery(%q) = nil, want error", q)
		}
	}
}

func TestValidateRefListAccepts(t *testing.T) {
	cases := []string{
		"Job 38:4-7",
		"Psalm 33:6",
		"Genesis 1:1; Revelation 22:21",
		"Song of Solomon 2:1",
		"1 Corinthians 13:1-3",
		// Only the first reference must name a book; ESV omits the book
		// on same-book continuations.
		"Psalm 33:6; 136:5",
		"John 3:16; 14:6",
		// en-dash, as ESV emits in cross-reference ranges.
		"Job 38:4–7",
	}
	for _, q := range cases {
		if err := ValidateRefList(q); err != nil {
			t.Errorf("ValidateRefList(%q) = %v, want nil", q, err)
		}
	}
}

func TestValidateRefListRejects(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"Booga 3:1",                         // unknown first book
		"Xyz 1:1",                           // unknown first book
		"12345",                             // no book
		"DROP TABLE",                        // no chapter tail → missing book
		"Job 3:1 <here>",                    // illegal character
		"漢 1:1",                             // illegal (non-ASCII) character
		strings.Repeat("Genesis 1:1; ", 30), // over the length cap
	}
	for _, q := range cases {
		if err := ValidateRefList(q); err == nil {
			t.Errorf("ValidateRefList(%q) = nil, want error", q)
		}
	}
}
