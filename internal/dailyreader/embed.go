// Package dailyreader looks up today's reading from one or more
// embedded markdown reading plans.
package dailyreader

import _ "embed"

//go:embed daily-reader.md
var bibleYearMarkdown []byte

//go:embed 2026_Hope_Bible_Reading_Plan.md
var hopeMarkdown []byte
