// Package canon is the translation-neutral 66-book Protestant canon
// shared across translation providers. Every translation supported by
// the app uses this canon and verse numbering — extending to canons
// with deuterocanonical books is a future-spec problem.
package canon

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Book is a single canonical book with its chapter count.
type Book struct {
	Name     string
	Chapters int
}

// Canon is the 66-book Protestant canon in canonical order.
var Canon = []Book{
	{"Genesis", 50}, {"Exodus", 40}, {"Leviticus", 27}, {"Numbers", 36}, {"Deuteronomy", 34},
	{"Joshua", 24}, {"Judges", 21}, {"Ruth", 4},
	{"1 Samuel", 31}, {"2 Samuel", 24}, {"1 Kings", 22}, {"2 Kings", 25},
	{"1 Chronicles", 29}, {"2 Chronicles", 36}, {"Ezra", 10}, {"Nehemiah", 13}, {"Esther", 10},
	{"Job", 42}, {"Psalms", 150}, {"Proverbs", 31}, {"Ecclesiastes", 12}, {"Song of Solomon", 8},
	{"Isaiah", 66}, {"Jeremiah", 52}, {"Lamentations", 5}, {"Ezekiel", 48}, {"Daniel", 12},
	{"Hosea", 14}, {"Joel", 3}, {"Amos", 9}, {"Obadiah", 1}, {"Jonah", 4}, {"Micah", 7},
	{"Nahum", 3}, {"Habakkuk", 3}, {"Zephaniah", 3}, {"Haggai", 2}, {"Zechariah", 14}, {"Malachi", 4},
	{"Matthew", 28}, {"Mark", 16}, {"Luke", 24}, {"John", 21}, {"Acts", 28},
	{"Romans", 16}, {"1 Corinthians", 16}, {"2 Corinthians", 13}, {"Galatians", 6},
	{"Ephesians", 6}, {"Philippians", 4}, {"Colossians", 4},
	{"1 Thessalonians", 5}, {"2 Thessalonians", 3}, {"1 Timothy", 6}, {"2 Timothy", 4},
	{"Titus", 3}, {"Philemon", 1}, {"Hebrews", 13}, {"James", 5},
	{"1 Peter", 5}, {"2 Peter", 3}, {"1 John", 5}, {"2 John", 1}, {"3 John", 1},
	{"Jude", 1}, {"Revelation", 22},
}

// bookAliases maps short forms (as they appear in daily-reader.md and
// other common references) to their canonical Canon names.
var bookAliases = map[string]string{
	"psalm":         "Psalms",
	"song of songs": "Song of Solomon",
	"1 sam.":        "1 Samuel",
	"2 sam.":        "2 Samuel",
	"1 chron.":      "1 Chronicles",
	"2 chron.":      "2 Chronicles",
	"lev.":          "Leviticus",
	"num.":          "Numbers",
	"deut.":         "Deuteronomy",
	"prov.":         "Proverbs",
	"eccl.":         "Ecclesiastes",
	"jer.":          "Jeremiah",
	"lam.":          "Lamentations",
	"zeph.":         "Zephaniah",
	"matt.":         "Matthew",
	"rev.":          "Revelation",
	"1 cor.":        "1 Corinthians",
	"2 cor.":        "2 Corinthians",
	"gal.":          "Galatians",
	"eph.":          "Ephesians",
	"philip.":       "Philippians",
	"col.":          "Colossians",
	"1 thess.":      "1 Thessalonians",
	"2 thess.":      "2 Thessalonians",
	"2 tim.":        "2 Timothy",
	"heb.":          "Hebrews",
}

// LookupBook returns the canonical Book for name, accepting case
// variants and known aliases. ok is false if name isn't in the canon.
// Shared with ValidateQuery via canonByName so the canon stays a
// single source of truth.
func LookupBook(name string) (Book, bool) {
	b, ok := canonByName[strings.ToLower(strings.TrimSpace(name))]
	return b, ok
}

// canonByName indexes Canon by lowercased book name plus accepted aliases.
var canonByName = func() map[string]Book {
	m := make(map[string]Book, len(Canon)*2)
	for _, b := range Canon {
		m[strings.ToLower(b.Name)] = b
	}
	for alias, name := range bookAliases {
		for _, b := range Canon {
			if b.Name == name {
				m[alias] = b
				break
			}
		}
	}
	return m
}()

// ValidateQuery checks that q matches an allow-listed reference shape:
//
//	"<book> <chapter>"
//	"<book> <chapter>-<chapter>"
//
// Verse-level references (`<book> <chapter>:<verse>` and verse ranges)
// were retired on 2026-05-07 to give NIV/ESV parity — YouVersion's
// passage endpoint doesn't accept verse-range USFM. The reader is now
// chapter-level only.
//
// It does not validate that chapters exist beyond
// "chapter <= book.Chapters". The provider's grammar enforces the
// rest. The point of this allow-list is to reject obvious garbage
// before consuming an upstream API call.
func ValidateQuery(q string) error {
	q = strings.TrimSpace(q)
	if q == "" {
		return errors.New("empty query")
	}
	if len(q) > 64 {
		return errors.New("query too long")
	}
	if strings.Contains(q, ":") {
		return errors.New("verse references are not supported; use \"<book> <chapter>\" or \"<book> <chapter>-<chapter>\"")
	}

	// Split book name from "chapter[-chapter]" tail.
	bookName, tail, ok := splitBookTail(q)
	if !ok {
		return errors.New("missing chapter")
	}

	book, ok := LookupBook(bookName)
	if !ok {
		return fmt.Errorf("unknown book: %q", bookName)
	}

	if strings.Contains(tail, "-") {
		startStr, endStr, _ := strings.Cut(tail, "-")
		startCh, err := parsePositiveInt(startStr)
		if err != nil {
			return fmt.Errorf("invalid chapter: %w", err)
		}
		endCh, err := parsePositiveInt(endStr)
		if err != nil {
			return fmt.Errorf("invalid chapter range end: %w", err)
		}
		if startCh < 1 || endCh > book.Chapters {
			return fmt.Errorf("chapter range %d-%d out of range for %s (1-%d)", startCh, endCh, book.Name, book.Chapters)
		}
		if endCh < startCh {
			return errors.New("chapter range end before start")
		}
		return nil
	}
	chapter, err := parsePositiveInt(tail)
	if err != nil {
		return fmt.Errorf("invalid chapter: %w", err)
	}
	if chapter < 1 || chapter > book.Chapters {
		return fmt.Errorf("chapter %d out of range for %s (1-%d)", chapter, book.Name, book.Chapters)
	}
	return nil
}

// ValidateRefList checks that q is a plausible list of cross-reference
// targets before it's forwarded to the ESV passage API by the cross-ref
// endpoint (internal/server/crossref.go). Unlike ValidateQuery — which
// is chapter-level and deliberately rejects verse syntax — these
// references come from ESV's own crossref markup and are verse-level
// (e.g. "Job 38:4-7; Psalm 33:6; Revelation 4:11"), so colons, commas,
// and ";"-separated lists are allowed.
//
// The point, as with ValidateQuery, is to reject obvious garbage before
// consuming an upstream call — not to fully parse every reference (ESV
// is the arbiter of reference grammar). It enforces three things:
//
//   - a length cap (lists concatenate several references);
//   - a restricted character set (letters, digits, and the punctuation
//     ESV emits: space . , : ; - and the en-dash "–");
//   - that the FIRST reference names a canonical book. ESV omits the
//     book name on same-book continuations ("Ps. 33:6; 136:5"), so only
//     the leading reference is checked for a book.
func ValidateRefList(q string) error {
	q = strings.TrimSpace(q)
	if q == "" {
		return errors.New("empty reference list")
	}
	if len(q) > 256 {
		return errors.New("reference list too long")
	}
	for _, r := range q {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == ' ' || r == '.' || r == ',' || r == ':' || r == ';' || r == '-' || r == '–':
		default:
			return fmt.Errorf("invalid character in reference list: %q", r)
		}
	}

	first := strings.TrimSpace(strings.SplitN(q, ";", 2)[0])
	bookName, _, ok := splitBookTail(first)
	if !ok {
		return errors.New("missing book in first reference")
	}
	if _, ok := LookupBook(bookName); !ok {
		return fmt.Errorf("unknown book: %q", bookName)
	}
	return nil
}

// splitBookTail separates a reference's book name from its trailing
// chapter/verse portion. The tail starts at the last space that is
// immediately followed by a digit, so multi-word and numbered book
// names ("Song of Solomon 1", "1 Corinthians 13:1") split correctly.
// ok is false when no chapter/verse tail is present.
func splitBookTail(ref string) (book, tail string, ok bool) {
	ref = strings.TrimSpace(ref)
	tailStart := -1
	for i := len(ref) - 1; i > 0; i-- {
		if ref[i] == ' ' && i+1 < len(ref) && ref[i+1] >= '0' && ref[i+1] <= '9' {
			tailStart = i
			break
		}
	}
	if tailStart <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(ref[:tailStart]), strings.TrimSpace(ref[tailStart+1:]), true
}

func parsePositiveInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty number")
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
	}
	return strconv.Atoi(s)
}
