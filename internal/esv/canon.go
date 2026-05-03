package esv

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

// Canon is the 66-book ESV canon in canonical order.
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
//	"<book> <chapter>:<verse>"
//	"<book> <chapter>:<verse>-<verse>"
//
// It does not validate that the chapter or verse exist beyond
// "chapter <= book.Chapters" and "verse >= 1". ESV's grammar enforces
// the rest. The point of this allow-list is to reject obvious garbage
// before consuming an ESV API call.
func ValidateQuery(q string) error {
	q = strings.TrimSpace(q)
	if q == "" {
		return errors.New("empty query")
	}
	if len(q) > 64 {
		return errors.New("query too long")
	}

	// Split book name from "chapter[:verse[-verse]]" tail.
	// The tail starts at the last space before a digit.
	tailStart := -1
	for i := len(q) - 1; i > 0; i-- {
		if q[i] == ' ' && i+1 < len(q) && q[i+1] >= '0' && q[i+1] <= '9' {
			tailStart = i
			break
		}
	}
	if tailStart <= 0 {
		return errors.New("missing chapter")
	}
	bookName := strings.TrimSpace(q[:tailStart])
	tail := strings.TrimSpace(q[tailStart+1:])

	book, ok := canonByName[strings.ToLower(bookName)]
	if !ok {
		return fmt.Errorf("unknown book: %q", bookName)
	}

	// tail = "<chapter>" or "<chapter>-<chapter>" or "<chapter>:<verse>" or "<chapter>:<verse>-<verse>"
	chapterStr, verseSpec, hasColon := strings.Cut(tail, ":")
	if !hasColon && strings.Contains(chapterStr, "-") {
		startStr, endStr, _ := strings.Cut(chapterStr, "-")
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
	chapter, err := parsePositiveInt(chapterStr)
	if err != nil {
		return fmt.Errorf("invalid chapter: %w", err)
	}
	if chapter < 1 || chapter > book.Chapters {
		return fmt.Errorf("chapter %d out of range for %s (1-%d)", chapter, book.Name, book.Chapters)
	}
	if !hasColon {
		return nil
	}

	startStr, endStr, hasDash := strings.Cut(verseSpec, "-")
	startVerse, err := parsePositiveInt(startStr)
	if err != nil {
		return fmt.Errorf("invalid verse: %w", err)
	}
	if startVerse < 1 {
		return errors.New("verse must be >= 1")
	}
	if !hasDash {
		return nil
	}
	endVerse, err := parsePositiveInt(endStr)
	if err != nil {
		return fmt.Errorf("invalid range end: %w", err)
	}
	if endVerse < startVerse {
		return errors.New("range end before start")
	}
	return nil
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
