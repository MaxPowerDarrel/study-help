package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"study-help/internal/auth"
	"study-help/internal/canon"
	"study-help/internal/scripture"
)

// passageHandler proxies validated passage requests to the active
// translation provider.
//
// Validation: q is checked against canon.ValidateQuery before any
// upstream call so malformed input returns 400 without consuming an
// upstream API call. Translation resolution: ?translation= wins;
// otherwise the signed-in user's account preference; otherwise the
// registry's default. Unknown ?translation= → 400.
func passageHandler(reg *scripture.Registry, counter *ESVCallCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if err := canon.ValidateQuery(q); err != nil {
			http.Error(w, "invalid q: "+err.Error(), http.StatusBadRequest)
			return
		}

		id, err := resolvePassageTranslation(r, reg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		provider, _ := reg.Get(id)

		opts := scripture.Options{
			IncludeHeadings:          boolParam(r, "include_headings", true),
			IncludeFootnotes:         boolParam(r, "include_footnotes", true),
			IncludeVerseNumbers:      boolParam(r, "include_verse_numbers", true),
			IncludePassageReferences: boolParam(r, "include_passage_references", true),
		}

		result, err := provider.Fetch(r.Context(), q, opts)
		if err != nil {
			switch {
			case errors.Is(err, scripture.ErrRateLimited):
				http.Error(w, "rate limited", http.StatusTooManyRequests)
			default:
				http.Error(w, "upstream error", http.StatusBadGateway)
			}
			log.Printf("passage q=%q translation=%s err=%v", q, id, err)
			return
		}

		// TODO(multi-translation): segment counter per translation
		// when a second provider lands; rename esv_api_calls_total.
		counter.Inc()
		log.Printf("passage q=%q translation=%s status=%d bytes=%d count=%d",
			q, id, result.Status, len(result.Body), counter.Value())

		ct := result.ContentType
		if ct == "" {
			ct = "application/json"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(result.Body)
	}
}

// resolvePassageTranslation picks the active translation for the
// passage endpoint. Explicit ?translation= wins; falls back to the
// signed-in user's account preference; falls back to the registry
// default. Unknown explicit IDs return an error mapped to 400.
func resolvePassageTranslation(r *http.Request, reg *scripture.Registry) (scripture.ID, error) {
	if q := r.URL.Query().Get("translation"); q != "" {
		id := scripture.ID(q)
		if !reg.Known(id) {
			return "", errBadTranslation(q)
		}
		return id, nil
	}
	if user, ok := auth.UserFromContext(r.Context()); ok && user != nil && user.Translation != "" {
		id := scripture.ID(user.Translation)
		if reg.Known(id) {
			return id, nil
		}
		// User has a translation that's no longer registered (provider
		// removed). Fall back to the default rather than 500ing.
	}
	return reg.Default(), nil
}

type translationError string

func (e translationError) Error() string { return string(e) }
func errBadTranslation(s string) error   { return translationError("unknown translation: " + s) }

// boolParam reads a query param as a bool, defaulting to fallback when
// absent or unparseable. Accepts "true"/"false"/"1"/"0".
func boolParam(r *http.Request, key string, fallback bool) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
