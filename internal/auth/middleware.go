package auth

import (
	"context"
	"errors"
	"net/http"
)

type ctxKey struct{}

var userCtxKey = ctxKey{}

// authPathSkipRefresh names paths that already manage Set-Cookie themselves
// (signin/signup issue, signout clears). The middleware skips its refresh
// pass for these to avoid double Set-Cookie headers; browsers honor only
// the last one.
var authPathSkipRefresh = map[string]bool{
	"/api/auth/signup":  true,
	"/api/auth/signin":  true,
	"/api/auth/signout": true,
}

// Middleware reads the session cookie, attaches the user to the request
// context if the session is valid, and refreshes Set-Cookie with the
// bumped Max-Age. On a missing/expired/unknown session it clears the
// cookie and continues without a user (handlers decide whether 401 is
// appropriate via RequireUser).
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := readSessionCookie(r)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		user, tok, err := s.Authenticate(r.Context(), raw)
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired) {
				clearSessionCookie(w, s.cfg.IsDev)
			}
			next.ServeHTTP(w, r)
			return
		}
		if !authPathSkipRefresh[r.URL.Path] {
			setSessionCookie(w, tok.Raw, tok.ExpiresAt, s.cfg.IsDev)
		}
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserFromContext returns the user attached by Middleware, if any.
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userCtxKey).(*User)
	return u, ok
}

// RequireUser wraps a handler so that requests without an authenticated
// user receive 401. Used by GET /api/auth/me; future highlight/note
// routes will use it too.
func RequireUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
