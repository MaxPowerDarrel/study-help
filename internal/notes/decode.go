package notes

import (
	"encoding/json"
	"errors"
	"net/http"
)

// decodeJSONBody enforces the MaxBodyBytes+1024 envelope cap and decodes
// the request body into v. On failure it writes a 400 response and
// returns false; callers should `return` immediately. The +1024 buffer
// covers the JSON envelope around a max-size body string — the precise
// len(body.Body) check still enforces the 16 KB cap on the field itself.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes+1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "body too large", http.StatusBadRequest)
			return false
		}
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}
