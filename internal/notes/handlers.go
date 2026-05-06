package notes

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
	StartOffset int    `json:"start_offset"`
	EndVerse    int    `json:"end_verse"`
	EndOffset   int    `json:"end_offset"`
	Body        string `json:"body"`
}

type patchBody struct {
	Body string `json:"body"`
}

// HandleList returns the user's notes for a passage, ordered by anchor.
// Translation comes from ?translation=, falling back to the user's
// account preference.
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
		items, err := listNotes(r.Context(), s.db, user.ID, string(translation), book.Name, chapter)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("notes list user_id=%d err=%v", user.ID, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, items)
	})
}

// HandleCreate inserts a new note. Multiple notes per range are allowed
// (specs/notes.md, 2026-05-04) so there is no overlap check.
// Translation comes from the body, falling back to the user's account
// preference.
func (s *Service) HandleCreate() http.HandlerFunc {
	return auth.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		// Cap the request body before decoding. The +1024 buffer covers
		// the JSON envelope around a max-size body string; the precise
		// len(body.Body) check below enforces the 16 KB cap on the field
		// itself.
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes+1024)
		var body createBody
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "body too large", http.StatusBadRequest)
				return
			}
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
		if len(body.Body) > MaxBodyBytes {
			http.Error(w, "body too large", http.StatusBadRequest)
			return
		}
		n := Note{
			Translation: string(translation),
			Book:        book.Name,
			Chapter:     body.Chapter,
			StartVerse:  body.StartVerse,
			StartOffset: body.StartOffset,
			EndVerse:    body.EndVerse,
			EndOffset:   body.EndOffset,
			Body:        body.Body,
		}
		saved, err := insertNote(r.Context(), s.db, user.ID, n, s.clock())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("notes create insert user_id=%d err=%v", user.ID, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, saved)
	})
}

// HandlePatch updates a note's body and bumps updated_at. Cross-user
// IDs return 404 (no enumeration), same posture as DELETE.
func (s *Service) HandlePatch() http.HandlerFunc {
	return auth.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes+1024)
		var body patchBody
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "body too large", http.StatusBadRequest)
				return
			}
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(body.Body) > MaxBodyBytes {
			http.Error(w, "body too large", http.StatusBadRequest)
			return
		}
		updated, err := updateNoteBody(r.Context(), s.db, user.ID, id, body.Body, s.clock())
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("notes patch user_id=%d id=%d err=%v", user.ID, id, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, updated)
	})
}

// HandleDelete removes a note by ID. IDs that don't exist or belong to
// another user return 404 to avoid ID enumeration.
func (s *Service) HandleDelete() http.HandlerFunc {
	return auth.RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := auth.UserFromContext(r.Context())
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := deleteNote(r.Context(), s.db, user.ID, id); err != nil {
			if errors.Is(err, ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("notes delete user_id=%d id=%d err=%v", user.ID, id, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
