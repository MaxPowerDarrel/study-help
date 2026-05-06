// Package httpx contains small HTTP helpers shared across handler
// packages: JSON response writing, a bad-request error constructor, and
// passage-query / translation parsers shared by /api/highlights and
// /api/notes.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"study-help/internal/canon"
	"study-help/internal/scripture"
)

// WriteJSON writes v to w with the given status and a JSON Content-Type.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// BadRequest returns an error whose message is suitable for a 400
// response body. Callers map it to http.StatusBadRequest via http.Error.
func BadRequest(msg string) error { return errors.New(msg) }

// ParsePassageQuery extracts and validates ?book=&chapter= from r against
// the canon. Errors carry user-safe messages for 400 responses.
func ParsePassageQuery(r *http.Request) (canon.Book, int, error) {
	bookParam := r.URL.Query().Get("book")
	chapterStr := r.URL.Query().Get("chapter")
	if bookParam == "" || chapterStr == "" {
		return canon.Book{}, 0, BadRequest("missing book or chapter")
	}
	chapter, err := strconv.Atoi(chapterStr)
	if err != nil {
		return canon.Book{}, 0, BadRequest("invalid chapter")
	}
	book, ok := canon.LookupBook(bookParam)
	if !ok {
		return canon.Book{}, 0, BadRequest("unknown book")
	}
	if chapter < 1 || chapter > book.Chapters {
		return canon.Book{}, 0, BadRequest("chapter out of range")
	}
	return book, chapter, nil
}

// ResolveTranslation picks the active translation for a request:
// supplied (body field or ?translation=) wins; falls back to the user's
// account preference. Unknown IDs return an error mapped to 400 by the
// caller.
func ResolveTranslation(supplied, fallback string, reg *scripture.Registry) (scripture.ID, error) {
	s := supplied
	if s == "" {
		s = fallback
	}
	id := scripture.ID(s)
	if !reg.Known(id) {
		return "", fmt.Errorf("unknown translation %q", s)
	}
	return id, nil
}
