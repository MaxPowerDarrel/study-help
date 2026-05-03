package esv

import "testing"

func TestValidateQueryAccepts(t *testing.T) {
	cases := []string{
		"John 3",
		"John 3:16",
		"John 3:1-21",
		"Genesis 1",
		"Revelation 22",
		"1 Samuel 17:45-50",
		"Song of Solomon 2:1",
		"Psalms 23",
		"Psalm 23",
		"3 John 1",
		"Malachi 4",
		"Obadiah 1",
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
		"John 3:0",
		"John 3:5-2",
		"John 3:abc",
		"<script>",
		"John 3; DROP TABLE",
	}
	for _, q := range cases {
		if err := ValidateQuery(q); err == nil {
			t.Errorf("ValidateQuery(%q) = nil, want error", q)
		}
	}
}
