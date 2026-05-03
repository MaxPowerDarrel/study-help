package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"study-help/internal/esv"
)

// passageHandler proxies validated passage requests to the ESV API.
//
// Validation: q is checked against esv.ValidateQuery before any upstream
// call so malformed input returns 400 without consuming an ESV API call.
// On success, the upstream response body is forwarded verbatim, the
// counter is incremented, and a structured log line is emitted.
func passageHandler(client *esv.Client, counter *ESVCallCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if err := esv.ValidateQuery(q); err != nil {
			http.Error(w, "invalid q: "+err.Error(), http.StatusBadRequest)
			return
		}

		opts := esv.Options{
			IncludeHeadings:          boolParam(r, "include_headings", true),
			IncludeFootnotes:         boolParam(r, "include_footnotes", true),
			IncludeVerseNumbers:      boolParam(r, "include_verse_numbers", true),
			IncludePassageReferences: boolParam(r, "include_passage_references", true),
		}

		result, err := client.Fetch(r.Context(), q, opts)
		if err != nil {
			switch {
			case errors.Is(err, esv.ErrRateLimited):
				http.Error(w, "rate limited", http.StatusTooManyRequests)
			default:
				http.Error(w, "upstream error", http.StatusBadGateway)
			}
			log.Printf("esv_call q=%q err=%v", q, err)
			return
		}

		counter.Inc()
		log.Printf("esv_call q=%q status=%d bytes=%d count=%d",
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
