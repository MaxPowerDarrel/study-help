package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"study-help/internal/scripture"
	"testing"
)

// signupReq builds a JSON request body for signup/signin.
func credsBody(email, password string) *bytes.Buffer {
	b, _ := json.Marshal(credentials{Email: email, Password: password})
	return bytes.NewBuffer(b)
}

// callViaMiddleware wires the auth middleware around a handler so the
// /me path can resolve the user from the cookie. RemoteAddr is set so
// the rate limiter has a usable key.
func callViaMiddleware(svc *Service, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	if req.RemoteAddr == "" {
		req.RemoteAddr = "1.2.3.4:1111"
	}
	w := httptest.NewRecorder()
	svc.Middleware(h).ServeHTTP(w, req)
	return w
}

func TestSignupReturns201AndSetsCookie(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", testPassword))
	req.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignup(rl)(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if cs := w.Result().Cookies(); len(cs) == 0 || cs[0].Name != CookieName {
		t.Fatalf("missing session cookie; cookies=%+v", cs)
	}
	var u User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if u.Email != "a@b.c" {
		t.Errorf("email %q", u.Email)
	}
}

func TestSignupHandlerRejectsShortPassword(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", "short"))
	req.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignup(rl)(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestSignupDuplicateEmailReturns409(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	req1 := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", testPassword))
	req1.RemoteAddr = "1.2.3.4:1111"
	svc.HandleSignup(rl)(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", testPassword))
	req2.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignup(rl)(w, req2)
	if w.Code != http.StatusConflict {
		t.Errorf("status %d, want 409", w.Code)
	}
}

func TestSigninWrongPasswordReturns401Ambiguous(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	signupReq := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", testPassword))
	signupReq.RemoteAddr = "1.2.3.4:1111"
	svc.HandleSignup(rl)(httptest.NewRecorder(), signupReq)

	signinReq := httptest.NewRequest(http.MethodPost, "/api/auth/signin", credsBody("a@b.c", "wrongpassword"))
	signinReq.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignin(rl)(w, signinReq)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid email or password") {
		t.Errorf("body should be ambiguous, got %q", w.Body.String())
	}
}

func TestSigninUnknownEmailReturns401Ambiguous(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", credsBody("nobody@example.com", testPassword))
	req.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignin(rl)(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid email or password") {
		t.Errorf("body %q", w.Body.String())
	}
}

func TestSigninRateLimitedByIP(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	// Burn through the per-IP burst with bad credentials. We expect
	// rlIPBurst attempts to be served (returning 401), and the next
	// one to be rejected with 429.
	for i := 0; i < rlIPBurst; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", credsBody(fmt.Sprintf("u%d@x.y", i), testPassword))
		req.RemoteAddr = "1.2.3.4:1111"
		w := httptest.NewRecorder()
		svc.HandleSignin(rl)(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("unexpected 429 on attempt %d", i+1)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", credsBody("more@x.y", testPassword))
	req.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignin(rl)(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("burst+1 status %d, want 429", w.Code)
	}
}

func TestSigninPerAccountLimiterTripsAcrossIPs(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	signupReq := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("victim@x.y", testPassword))
	signupReq.RemoteAddr = "1.2.3.4:1111"
	svc.HandleSignup(rl)(httptest.NewRecorder(), signupReq)

	// Burn the per-account bucket with wrong-password attempts from
	// distinct IPs (so the per-IP bucket doesn't trip first).
	for i := 0; i < rlAccountBurst; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", credsBody("victim@x.y", "wrongpassword"))
		req.RemoteAddr = fmt.Sprintf("9.9.9.%d:1111", i+1)
		w := httptest.NewRecorder()
		svc.HandleSignin(rl)(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d, want 401", i+1, w.Code)
		}
	}
	// Next attempt (from yet another IP) should trip the per-account
	// limiter even though that IP's bucket is fresh.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", credsBody("victim@x.y", "wrongpassword"))
	req.RemoteAddr = "9.9.9.99:1111"
	w := httptest.NewRecorder()
	svc.HandleSignin(rl)(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("burst+1 status %d, want 429", w.Code)
	}
}

func TestSignoutWithoutCookieReturns204(t *testing.T) {
	svc, _ := newTestService(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signout", nil)
	w := httptest.NewRecorder()
	svc.HandleSignout()(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status %d, want 204", w.Code)
	}
}

func TestSignoutWithUnknownCookieReturns204(t *testing.T) {
	svc, _ := newTestService(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signout", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"})
	w := httptest.NewRecorder()
	svc.HandleSignout()(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status %d, want 204", w.Code)
	}
}

func TestSignoutDeletesValidSessionAndIsIdempotent(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()
	signupReq := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", testPassword))
	signupReq.RemoteAddr = "1.2.3.4:1111"
	signupResp := httptest.NewRecorder()
	svc.HandleSignup(rl)(signupResp, signupReq)
	cookie := signupResp.Result().Cookies()[0]

	signoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/signout", nil)
	signoutReq.AddCookie(cookie)
	signoutResp := httptest.NewRecorder()
	svc.HandleSignout()(signoutResp, signoutReq)
	if signoutResp.Code != http.StatusNoContent {
		t.Errorf("signout status %d, want 204", signoutResp.Code)
	}
	// Re-issuing the signout with the same (now invalid) cookie still
	// returns 204.
	signoutReq2 := httptest.NewRequest(http.MethodPost, "/api/auth/signout", nil)
	signoutReq2.AddCookie(cookie)
	signoutResp2 := httptest.NewRecorder()
	svc.HandleSignout()(signoutResp2, signoutReq2)
	if signoutResp2.Code != http.StatusNoContent {
		t.Errorf("repeat signout status %d, want 204", signoutResp2.Code)
	}
}

func TestMeGuestReturns401(t *testing.T) {
	svc, _ := newTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleMe(), req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", w.Code)
	}
}

func TestMeSignedInReturnsUser(t *testing.T) {
	svc, _ := newTestService(t)
	rl := NewLimiter()

	signupReq := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody("a@b.c", testPassword))
	signupReq.RemoteAddr = "1.2.3.4:1111"
	signupResp := httptest.NewRecorder()
	svc.HandleSignup(rl)(signupResp, signupReq)
	cookie := signupResp.Result().Cookies()[0]

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.AddCookie(cookie)
	meReq.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleMe(), meReq)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var u User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if u.Email != "a@b.c" {
		t.Errorf("email %q", u.Email)
	}
	if u.Translation != string(scripture.ESV) {
		t.Errorf("translation %q, want ESV", u.Translation)
	}
}

// fakeProvider is the minimum to register an alternative ID for the
// translation endpoint tests; mirrors scripture.fakeProvider but lives
// here so we don't export the test helper.
type fakeProvider struct{ id scripture.ID }

func (f *fakeProvider) ID() scripture.ID    { return f.id }
func (f *fakeProvider) DisplayName() string { return string(f.id) }
func (f *fakeProvider) Fetch(_ context.Context, _ string, _ scripture.Options) (*scripture.Result, error) {
	return nil, nil
}

func newTestRegistry() *scripture.Registry {
	return scripture.NewRegistry(scripture.ESV,
		&fakeProvider{id: scripture.ESV},
		&fakeProvider{id: scripture.ID("KJV")},
	)
}

func signupAndGetCookie(t *testing.T, svc *Service, email string) *http.Cookie {
	t.Helper()
	rl := NewLimiter()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", credsBody(email, testPassword))
	req.RemoteAddr = "1.2.3.4:1111"
	w := httptest.NewRecorder()
	svc.HandleSignup(rl)(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("signup status %d, body=%s", w.Code, w.Body.String())
	}
	return w.Result().Cookies()[0]
}

func TestUpdateMeGuestReturns401(t *testing.T) {
	svc, _ := newTestService(t)
	reg := newTestRegistry()
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{"translation":"ESV"}`))
	req.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleUpdateMe(reg), req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", w.Code)
	}
}

func TestUpdateMeUnknownTranslationReturns400(t *testing.T) {
	svc, _ := newTestService(t)
	reg := newTestRegistry()
	cookie := signupAndGetCookie(t, svc, "a@b.c")

	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{"translation":"BOGUS"}`))
	req.AddCookie(cookie)
	req.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleUpdateMe(reg), req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestUpdateMeHappyPath(t *testing.T) {
	svc, _ := newTestService(t)
	reg := newTestRegistry()
	cookie := signupAndGetCookie(t, svc, "a@b.c")

	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{"translation":"KJV"}`))
	req.AddCookie(cookie)
	req.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleUpdateMe(reg), req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}
	var u User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if u.Translation != "KJV" {
		t.Errorf("translation %q, want KJV", u.Translation)
	}
}

func TestUpdateMeAbsentFieldIsNoop(t *testing.T) {
	svc, _ := newTestService(t)
	reg := newTestRegistry()
	cookie := signupAndGetCookie(t, svc, "a@b.c")

	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{}`))
	req.AddCookie(cookie)
	req.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleUpdateMe(reg), req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}
	var u User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if u.Translation != string(scripture.ESV) {
		t.Errorf("translation %q, want ESV (no-op)", u.Translation)
	}
}

func TestUpdateMeUnknownFieldReturns400(t *testing.T) {
	svc, _ := newTestService(t)
	reg := newTestRegistry()
	cookie := signupAndGetCookie(t, svc, "a@b.c")

	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", strings.NewReader(`{"theme":"dark"}`))
	req.AddCookie(cookie)
	req.RemoteAddr = "1.2.3.4:1111"
	w := callViaMiddleware(svc, svc.HandleUpdateMe(reg), req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400 (DisallowUnknownFields)", w.Code)
	}
}
