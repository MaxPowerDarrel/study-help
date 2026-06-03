package server

import (
	"errors"
	"log"
	"net/http"

	"study-help/internal/canon"
	"study-help/internal/scripture"
)

// crossrefHandler looks up the verse text behind a cross-reference
// marker for the cross-ref popover. The marker's referenced verses
// (e.g. "Job 38:4-7; Psalm 33:6") arrive in ?q= and are validated with
// canon.ValidateRefList — the verse-level allow-list, since these refs
// are verse-granular unlike the chapter-level reader.
//
// It is ESV-only by construction: cross-reference markers only exist in
// ESV-rendered passages (YouVersion/NIV emits none), so there is no
// translation to resolve. The response is the same {canonical, passages}
// envelope /api/passage returns, so the SPA renders it with the shared
// .passage styling.
func crossrefHandler(reg *scripture.Registry, counter *ESVCallCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if err := canon.ValidateRefList(q); err != nil {
			http.Error(w, "invalid q: "+err.Error(), http.StatusBadRequest)
			return
		}

		provider, err := reg.Get(scripture.ESV)
		if err != nil {
			http.Error(w, "cross-references unavailable", http.StatusBadGateway)
			return
		}

		// Headings/footnotes off and crossrefs off so the popover shows
		// plain verse text without nesting more markers; verse numbers and
		// the passage-reference label orient the reader inside the popover.
		opts := scripture.Options{
			IncludeHeadings:          false,
			IncludeFootnotes:         false,
			IncludeVerseNumbers:      true,
			IncludePassageReferences: true,
			IncludeCrossReferences:   false,
		}

		result, err := provider.Fetch(r.Context(), q, opts)
		if err != nil {
			switch {
			case errors.Is(err, scripture.ErrRateLimited):
				http.Error(w, "rate limited", http.StatusTooManyRequests)
			default:
				http.Error(w, "upstream error", http.StatusBadGateway)
			}
			log.Printf("crossref q=%q err=%v", q, err)
			return
		}

		counter.Inc()
		log.Printf("crossref q=%q status=%d bytes=%d count=%d",
			q, result.Status, len(result.Body), counter.Value())

		ct := result.ContentType
		if ct == "" {
			ct = "application/json"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(result.Body)
	}
}
