package highlights

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"study-help/internal/auth"
	"study-help/internal/canon"
	"study-help/internal/httpx"
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
		translation, err := httpx.ResolveTranslation(r.URL.Query().Get("translation"), user.Translation, s.reg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		book, chapter, err := httpx.ParsePassageQuery(r)
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
		httpx.WriteJSON(w, http.StatusOK, items)
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
		translation, err := httpx.ResolveTranslation(body.Translation, user.Translation, s.reg)
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
		httpx.WriteJSON(w, http.StatusCreated, saved)
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
