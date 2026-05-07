package canon

import "testing"

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
