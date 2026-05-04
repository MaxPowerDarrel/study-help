package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const testPassword = "twelvecharspw"

func newTestService(t *testing.T) (*Service, func(time.Time)) {
	t.Helper()
	d := newTestDB(t)
	clk := &fakeClock{now: time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)}
	svc := NewService(d, Config{
		IsDev:      true,
		BcryptCost: bcrypt.MinCost,
		SessionTTL: 30 * 24 * time.Hour,
		Clock:      clk.Now,
	})
	return svc, func(t time.Time) { clk.set(t) }
}

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) set(t time.Time)         { c.now = t }
func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func TestSignupCreatesUserAndSession(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	user, tok, err := svc.Signup(ctx, "Alice@Example.com ", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email not normalized: %q", user.Email)
	}
	if tok.Raw == "" {
		t.Error("empty session token")
	}
	if tok.ExpiresAt.IsZero() {
		t.Error("zero expires_at")
	}
}

func TestSignupRejectsDuplicateEmail(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, _, err := svc.Signup(ctx, "a@b.c", testPassword); err != nil {
		t.Fatalf("first Signup: %v", err)
	}
	_, _, err := svc.Signup(ctx, "A@B.C", testPassword)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("dup signup: got %v, want ErrEmailTaken", err)
	}
}

func TestSignupRejectsShortPassword(t *testing.T) {
	svc, _ := newTestService(t)
	_, _, err := svc.Signup(context.Background(), "a@b.c", "short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("short pw: got %v, want ErrWeakPassword", err)
	}
}

func TestSigninWithValidCreds(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, _, err := svc.Signup(ctx, "a@b.c", testPassword); err != nil {
		t.Fatalf("Signup: %v", err)
	}
	user, tok, err := svc.Signin(ctx, "A@B.C", testPassword)
	if err != nil {
		t.Fatalf("Signin: %v", err)
	}
	if user.Email != "a@b.c" {
		t.Errorf("email: %q", user.Email)
	}
	if tok.Raw == "" {
		t.Error("empty token")
	}
}

func TestSigninWrongPasswordReturnsAmbiguousError(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, _, err := svc.Signup(ctx, "a@b.c", testPassword); err != nil {
		t.Fatalf("Signup: %v", err)
	}
	_, _, err := svc.Signin(ctx, "a@b.c", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong pw: got %v, want ErrInvalidCredentials", err)
	}
}

func TestSigninUnknownEmailReturnsAmbiguousError(t *testing.T) {
	svc, _ := newTestService(t)
	_, _, err := svc.Signin(context.Background(), "nobody@example.com", testPassword)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email: got %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticateReturnsUserAndBumpsSlidingWindow(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, tok, err := svc.Signup(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	originalExpiry := tok.ExpiresAt

	// Advance the clock by a day; Authenticate should bump expires_at.
	clk := svc.now().Add(24 * time.Hour)
	svc.now = func() time.Time { return clk }

	user, refreshed, err := svc.Authenticate(ctx, tok.Raw)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.Email != "a@b.c" {
		t.Errorf("email: %q", user.Email)
	}
	if !refreshed.ExpiresAt.After(originalExpiry) {
		t.Errorf("expiry not bumped: %v <= %v", refreshed.ExpiresAt, originalExpiry)
	}
	if refreshed.Raw != tok.Raw {
		t.Error("Authenticate should not rotate the cookie value")
	}
}

func TestAuthenticateUsesHashLookup(t *testing.T) {
	// Verify that the row's token_hash is the sha256, not the raw token.
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, tok, err := svc.Signup(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	var stored []byte
	if err := svc.db.QueryRowContext(ctx, `SELECT token_hash FROM sessions LIMIT 1`).Scan(&stored); err != nil {
		t.Fatalf("query: %v", err)
	}
	if string(stored) == tok.Raw {
		t.Fatal("stored hash equals raw token; should be sha256(raw bytes)")
	}
	// And Authenticate still works using the raw token.
	if _, _, err := svc.Authenticate(ctx, tok.Raw); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
}

func TestAuthenticateRejectsExpiredSession(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, tok, err := svc.Signup(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	// Advance past the 30-day TTL.
	future := svc.now().Add(31 * 24 * time.Hour)
	svc.now = func() time.Time { return future }

	_, _, err = svc.Authenticate(ctx, tok.Raw)
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expired: got %v, want ErrSessionExpired", err)
	}
	// The expired row should have been deleted as a side-effect.
	var count int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expired session not cleaned up; %d rows remain", count)
	}
}

func TestAuthenticateRejectsUnknownToken(t *testing.T) {
	svc, _ := newTestService(t)
	_, _, err := svc.Authenticate(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("unknown token: got %v, want ErrSessionNotFound", err)
	}
}

func TestSignoutDeletesSession(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	_, tok, err := svc.Signup(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if err := svc.Signout(ctx, tok.Raw); err != nil {
		t.Fatalf("Signout: %v", err)
	}
	if _, _, err := svc.Authenticate(ctx, tok.Raw); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("post-signout Authenticate: got %v, want ErrSessionNotFound", err)
	}
}

func TestSignoutOnlyAffectsCurrentSession(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	_, tokA, err := svc.Signup(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	_, tokB, err := svc.Signin(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signin: %v", err)
	}
	if err := svc.Signout(ctx, tokA.Raw); err != nil {
		t.Fatalf("Signout A: %v", err)
	}
	// Session B must still be valid.
	if _, _, err := svc.Authenticate(ctx, tokB.Raw); err != nil {
		t.Fatalf("Authenticate B after Signout A: %v", err)
	}
	// Session A must be gone.
	if _, _, err := svc.Authenticate(ctx, tokA.Raw); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Authenticate A: got %v, want ErrSessionNotFound", err)
	}
}

func TestSignoutIsIdempotent(t *testing.T) {
	svc, _ := newTestService(t)
	if err := svc.Signout(context.Background(), ""); err != nil {
		t.Fatalf("Signout empty: %v", err)
	}
	if err := svc.Signout(context.Background(), "garbage!!!"); err != nil {
		t.Fatalf("Signout garbage: %v", err)
	}
	if err := svc.Signout(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"); err != nil {
		t.Fatalf("Signout unknown valid base64url: %v", err)
	}
}

func TestForeignKeyCascadeDeletesSessionsOnUserDelete(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	user, _, err := svc.Signup(ctx, "a@b.c", testPassword)
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, user.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	var count int
	if err := svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id = ?`, user.ID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("orphan sessions: %d (want 0; FK cascade should have cleaned them)", count)
	}
}
