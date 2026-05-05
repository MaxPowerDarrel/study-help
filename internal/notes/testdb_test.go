package notes

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"study-help/internal/auth"
	"study-help/internal/config"
	"study-help/internal/db"
)

// newTestDB opens a temp-file SQLite via the production db.Open path so
// the real migrations run (including 00004_notes.sql).
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{
		DatabaseURL:   filepath.Join(dir, "test.db"),
		SessionSecret: "test",
		ESVAPIKey:     "test",
	}
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// newTestStack returns a paired notes + auth service backed by the same
// database, so handler tests can sign a user up and exercise the real
// auth.Middleware path.
func newTestStack(t *testing.T) (*Service, *auth.Service, *sql.DB) {
	t.Helper()
	d := newTestDB(t)
	authSvc := auth.NewService(d, auth.Config{
		IsDev:      true,
		BcryptCost: bcrypt.MinCost,
		SessionTTL: 30 * 24 * time.Hour,
	})
	nSvc := NewService(d, Config{})
	return nSvc, authSvc, d
}

const testPassword = "very-long-password-12"

// signupUser creates a user via the auth service and returns the user
// plus their session token (the value to put in a session cookie).
func signupUser(t *testing.T, authSvc *auth.Service, email string) (*auth.User, string) {
	t.Helper()
	user, tok, err := authSvc.Signup(context.Background(), email, testPassword)
	if err != nil {
		t.Fatalf("signup %q: %v", email, err)
	}
	return user, tok.Raw
}
