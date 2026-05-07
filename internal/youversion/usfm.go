package youversion

import (
	"strings"
	"study-help/internal/canon"
)

// usfmByName maps the 66-book Protestant canon (in canonical English
// names, lower-cased) to USFM 3.0 book identifiers used by YouVersion's
// passage endpoint. Source of truth: https://ubsicap.github.io/usfm/.
var usfmByName = map[string]string{
	// Old Testament
	"genesis": "GEN", "exodus": "EXO", "leviticus": "LEV", "numbers": "NUM",
	"deuteronomy": "DEU", "joshua": "JOS", "judges": "JDG", "ruth": "RUT",
	"1 samuel": "1SA", "2 samuel": "2SA", "1 kings": "1KI", "2 kings": "2KI",
	"1 chronicles": "1CH", "2 chronicles": "2CH", "ezra": "EZR",
	"nehemiah": "NEH", "esther": "EST", "job": "JOB", "psalms": "PSA",
	"proverbs": "PRO", "ecclesiastes": "ECC", "song of solomon": "SNG",
	"isaiah": "ISA", "jeremiah": "JER", "lamentations": "LAM",
	"ezekiel": "EZK", "daniel": "DAN", "hosea": "HOS", "joel": "JOL",
	"amos": "AMO", "obadiah": "OBA", "jonah": "JON", "micah": "MIC",
	"nahum": "NAM", "habakkuk": "HAB", "zephaniah": "ZEP", "haggai": "HAG",
	"zechariah": "ZEC", "malachi": "MAL",
	// New Testament
	"matthew": "MAT", "mark": "MRK", "luke": "LUK", "john": "JHN",
	"acts": "ACT", "romans": "ROM", "1 corinthians": "1CO",
	"2 corinthians": "2CO", "galatians": "GAL", "ephesians": "EPH",
	"philippians": "PHP", "colossians": "COL",
	"1 thessalonians": "1TH", "2 thessalonians": "2TH",
	"1 timothy": "1TI", "2 timothy": "2TI", "titus": "TIT",
	"philemon": "PHM", "hebrews": "HEB", "james": "JAS",
	"1 peter": "1PE", "2 peter": "2PE", "1 john": "1JN",
	"2 john": "2JN", "3 john": "3JN", "jude": "JUD", "revelation": "REV",
}

// usfmBook returns the 3-character USFM book code for a book name.
// Aliases ("Psalm", "Song of Songs", abbreviations) are resolved
// through canon.LookupBook so callers can pass the same names that
// canon.ValidateQuery accepts. The empty string indicates an unknown
// book.
func usfmBook(name string) string {
	b, ok := canon.LookupBook(name)
	if !ok {
		return ""
	}
	return usfmByName[strings.ToLower(b.Name)]
}
