// Package dailyreader looks up today's reading in the Bible-in-One-Year plan
// shipped as an embedded markdown table.
package dailyreader

import _ "embed"

//go:embed daily-reader.md
var planMarkdown []byte
