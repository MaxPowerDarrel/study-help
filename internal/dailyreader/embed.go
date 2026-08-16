// Package dailyreader looks up today's reading from one or more
// embedded markdown reading plans.
package dailyreader

import _ "embed"

//go:embed daily-reader.md
var bibleYearMarkdown []byte

//go:embed 2026_Hope_Bible_Reading_Plan.md
var hopeMarkdown []byte

// study-plan.md is generated; regenerate it after changing the plan
// shape in gen/main.go.
//
//go:generate go run ./gen

//go:embed study-plan.md
var studyMarkdown []byte
