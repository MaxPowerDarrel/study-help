package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddlewareAttachesUserOnValidCookie(t *testing.T) {
	svc, _ := newTestService(t)
	user, tok, err := svc.Signup(context.Background(), "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	var seen *User
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := UserFromContext(r.Context())
		seen = u
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/passage", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok.Raw})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if seen == nil || seen.ID != user.ID {
		t.Fatalf("user not attached: %+v", seen)
	}
}

func TestMiddlewarePassesThroughWithoutCookie(t *testing.T) {
	svc, _ := newTestService(t)
	called := false
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := UserFromContext(r.Context()); ok {
			t.Error("user attached on cookie-less request")
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/passage", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Fatal("downstream handler not called")
	}
}

func TestMiddlewareClearsCookieOnUnknownSession(t *testing.T) {
	svc, _ := newTestService(t)
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/api/passage", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie clearing the session, got none")
	}
	if cookies[0].MaxAge >= 0 {
		t.Errorf("expected MaxAge<0, got %d", cookies[0].MaxAge)
	}
}

func TestMiddlewareRefreshesCookieMaxAge(t *testing.T) {
	svc, _ := newTestService(t)
	_, tok, err := svc.Signup(context.Background(), "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	originalExpiry := tok.ExpiresAt

	// Advance the clock so the refresh produces a visibly later expiry.
	advance := svc.now().Add(2 * time.Hour)
	svc.now = func() time.Time { return advance }

	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/api/passage", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok.Raw})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refreshed Set-Cookie")
	}
	if !cookies[0].Expires.After(originalExpiry) {
		t.Errorf("refresh did not bump Expires: %v <= %v", cookies[0].Expires, originalExpiry)
	}
}

func TestMiddlewareSkipsRefreshOnAuthPaths(t *testing.T) {
	svc, _ := newTestService(t)
	_, tok, err := svc.Signup(context.Background(), "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	h := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signout", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: tok.Raw})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if cs := w.Result().Cookies(); len(cs) != 0 {
		t.Errorf("expected no Set-Cookie from middleware on auth path, got %d", len(cs))
	}
}

func TestRequireUserReturns401WhenAbsent(t *testing.T) {
	h := RequireUser(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not run")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", w.Code)
	}
}
