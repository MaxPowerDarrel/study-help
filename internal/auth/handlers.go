package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"study-help/internal/scripture"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decodeCredentials(r *http.Request) (credentials, error) {
	var c credentials
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return credentials{}, err
	}
	return c, nil
}

func writeUserJSON(w http.ResponseWriter, status int, u *User) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(u)
}

// HandleSignup creates an account and returns 201 + Set-Cookie + user.
// Per-IP rate limited; per-account limit doesn't apply (no account yet).
func (s *Service) HandleSignup(rl *Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.AllowIP(r.RemoteAddr) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			log.Printf("signup ip=%s outcome=rate_limited", hostOf(r.RemoteAddr))
			return
		}
		creds, err := decodeCredentials(r)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		user, tok, err := s.Signup(r.Context(), creds.Email, creds.Password)
		switch {
		case errors.Is(err, ErrWeakPassword):
			http.Error(w, "password must be at least 12 characters", http.StatusBadRequest)
			log.Printf("signup email=%q outcome=rejected reason=weak_password", normalizeEmail(creds.Email))
			return
		case errors.Is(err, ErrEmailTaken):
			http.Error(w, "email already in use", http.StatusConflict)
			log.Printf("signup email=%q outcome=rejected reason=duplicate", normalizeEmail(creds.Email))
			return
		case err != nil:
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("signup email=%q outcome=error err=%v", normalizeEmail(creds.Email), err)
			return
		}
		setSessionCookie(w, tok.Raw, tok.ExpiresAt, s.cfg.IsDev)
		writeUserJSON(w, http.StatusCreated, user)
		log.Printf("signup email=%q outcome=success user_id=%d", user.Email, user.ID)
	}
}

// HandleSignin validates credentials and returns 200 + Set-Cookie + user.
// Per-IP and per-account rate limited; on credential mismatch consume a
// per-account token.
func (s *Service) HandleSignin(rl *Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.AllowIP(r.RemoteAddr) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			log.Printf("signin ip=%s outcome=rate_limited reason=ip", hostOf(r.RemoteAddr))
			return
		}
		creds, err := decodeCredentials(r)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		email := normalizeEmail(creds.Email)
		if !rl.AllowAccount(email) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			log.Printf("signin email=%q outcome=rate_limited reason=account", email)
			return
		}
		user, tok, err := s.Signin(r.Context(), creds.Email, creds.Password)
		if err != nil {
			if errors.Is(err, ErrInvalidCredentials) {
				rl.RecordAccountFailure(email)
				http.Error(w, "invalid email or password", http.StatusUnauthorized)
				log.Printf("signin email=%q outcome=failed reason=invalid_credentials", email)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			log.Printf("signin email=%q outcome=error err=%v", email, err)
			return
		}
		setSessionCookie(w, tok.Raw, tok.ExpiresAt, s.cfg.IsDev)
		writeUserJSON(w, http.StatusOK, user)
		log.Printf("signin email=%q outcome=success user_id=%d", user.Email, user.ID)
	}
}

// HandleSignout is idempotent: always returns 204 and clears the cookie.
// If the cookie names a valid session, the row is deleted.
func (s *Service) HandleSignout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := readSessionCookie(r)
		if err := s.Signout(r.Context(), raw); err != nil {
			log.Printf("signout outcome=error err=%v", err)
		}
		clearSessionCookie(w, s.cfg.IsDev)
		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleMe returns the authenticated user, or 401. The middleware has
// already attached the user (and refreshed the cookie) if the session
// is valid.
func (s *Service) HandleMe() http.HandlerFunc {
	return RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		writeUserJSON(w, http.StatusOK, user)
	})
}

// HandleUpdateMe applies a partial update to the authenticated user.
// Currently the only mutable field is translation; absent fields are
// no-ops, so the same endpoint can grow new preferences without forcing
// the client to send everything every call.
func (s *Service) HandleUpdateMe(reg *scripture.Registry) http.HandlerFunc {
	return RequireUser(func(w http.ResponseWriter, r *http.Request) {
		user, _ := UserFromContext(r.Context())
		var body struct {
			Translation *string `json:"translation"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body.Translation != nil {
			id := scripture.ID(*body.Translation)
			if !reg.Known(id) {
				http.Error(w, "unknown translation", http.StatusBadRequest)
				return
			}
			updated, err := updateUserTranslation(r.Context(), s.db, user.ID, string(id), s.now().UTC())
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				log.Printf("update_me user_id=%d outcome=error err=%v", user.ID, err)
				return
			}
			user = updated
		}
		writeUserJSON(w, http.StatusOK, user)
	})
}
