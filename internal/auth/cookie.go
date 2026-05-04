package auth

import (
	"net/http"
	"time"
)

// CookieName is the session cookie name. Exported so tests in other
// packages (and the future client-side fetch wrapper) can reference it.
const CookieName = "sh_session"

// setSessionCookie writes Set-Cookie with the canonical attributes.
// Path "/", HttpOnly, SameSite=Lax, Secure unless dev. Expires and
// Max-Age mirror the session row's expires_at (sliding window).
func setSessionCookie(w http.ResponseWriter, raw string, expiresAt time.Time, isDev bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   !isDev,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

// clearSessionCookie expires the cookie immediately. Attributes must
// match setSessionCookie or browsers may leave a stale cookie behind.
func clearSessionCookie(w http.ResponseWriter, isDev bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   !isDev,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// readSessionCookie returns the cookie value, or "" if absent/malformed.
func readSessionCookie(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
