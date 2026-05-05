package highlights

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"study-help/internal/auth"
	"study-help/internal/canon"
	"study-help/internal/scripture"
)

type createBody struct {
	Translation string `json:"translation"`
	Book        string `json:"book"`
	Chapter     int    `json:"chapter"`
	StartVerse  int    `json:"start_verse"`
	StartOffset int    `json:"start_char_offset"`
	EndVerse    int    `json:"end_verse"`
	EndOffset   int    `json:"end_char_offset"`
}

// HandleList returns the user's highlights for a passage in the active
// translation. Translation comes from ?translation=, falling back to
// the user's account preference.
func (s *Service) HandleList() http.HandlerFunc {
	return auth.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		translation, err := resolveTranslation(r.URL.Query().Get("translation"), user, s.reg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		book, chapter, err := parsePassageQuery(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		items, err := listHighlights(r.Context(), s.db, user.ID, string(translation), book.Name, chapter)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("highlights list user_id=%d err=%v", user.ID, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	})
}

// HandleCreate inserts a new highlight or returns 409 on overlap.
// Translation comes from the body, falling back to the user's account
// preference; overlap is scoped per-translation.
func (s *Service) HandleCreate() http.HandlerFunc {
	return auth.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		var body createBody
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		translation, err := resolveTranslation(body.Translation, user, s.reg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		book, ok := canon.LookupBook(body.Book)
		if !ok {
			http.Error(w, "unknown book", http.StatusBadRequest)
			return
		}
		if body.Chapter < 1 || body.Chapter > book.Chapters {
			http.Error(w, "chapter out of range", http.StatusBadRequest)
			return
		}
		if body.StartVerse < 1 || body.EndVerse < body.StartVerse {
			http.Error(w, "invalid verse range", http.StatusBadRequest)
			return
		}
		if body.StartOffset < 0 || body.EndOffset < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		if body.StartVerse == body.EndVerse && body.EndOffset <= body.StartOffset {
			http.Error(w, "empty range", http.StatusBadRequest)
			return
		}
		h := Highlight{
			Translation: string(translation),
			Book:        book.Name,
			Chapter:     body.Chapter,
			StartVerse:  body.StartVerse,
			StartOffset: body.StartOffset,
			EndVerse:    body.EndVerse,
			EndOffset:   body.EndOffset,
		}
		existing, err := listHighlights(r.Context(), s.db, user.ID, string(translation), book.Name, body.Chapter)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("highlights create lookup user_id=%d err=%v", user.ID, err)
			return
		}
		for _, e := range existing {
			if rangesOverlap(h.StartVerse, h.StartOffset, h.EndVerse, h.EndOffset,
				e.StartVerse, e.StartOffset, e.EndVerse, e.EndOffset) {
				http.Error(w, "overlaps existing highlight", http.StatusConflict)
				return
			}
		}
		saved, err := insertHighlight(r.Context(), s.db, user.ID, h, s.clock())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("highlights create insert user_id=%d err=%v", user.ID, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	})
}

// HandleDelete removes a highlight by ID. IDs that don't exist or
// belong to another user return 404 to avoid ID enumeration.
func (s *Service) HandleDelete() http.HandlerFunc {
	return auth.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := deleteHighlight(r.Context(), s.db, user.ID, id); err != nil {
			if errors.Is(err, ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("highlights delete user_id=%d id=%d err=%v", user.ID, id, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// resolveTranslation picks the active translation for a request:
// supplied (body field or ?translation=) wins; falls back to the user's
// account preference. Unknown IDs return an error mapped to 400 by the
// caller.
func resolveTranslation(supplied string, user *auth.User, reg *scripture.Registry) (scripture.ID, error) {
	s := supplied
	if s == "" {
		s = user.Translation
	}
	id := scripture.ID(s)
	if !reg.Known(id) {
		return "", fmt.Errorf("unknown translation %q", s)
	}
	return id, nil
}

func parsePassageQuery(r *http.Request) (canon.Book, int, error) {
	bookParam := r.URL.Query().Get("book")
	chapterStr := r.URL.Query().Get("chapter")
	if bookParam == "" || chapterStr == "" {
		return canon.Book{}, 0, errBadRequest("missing book or chapter")
	}
	chapter, err := strconv.Atoi(chapterStr)
	if err != nil {
		return canon.Book{}, 0, errBadRequest("invalid chapter")
	}
	book, ok := canon.LookupBook(bookParam)
	if !ok {
		return canon.Book{}, 0, errBadRequest("unknown book")
	}
	if chapter < 1 || chapter > book.Chapters {
		return canon.Book{}, 0, errBadRequest("chapter out of range")
	}
	return book, chapter, nil
}

type httpError string

func (e httpError) Error() string  { return string(e) }
func errBadRequest(s string) error { return httpError(s) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
