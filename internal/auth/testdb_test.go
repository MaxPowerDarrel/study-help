package auth

import (
	"database/sql"
	"path/filepath"
	"testing"

	"study-help/internal/config"
	"study-help/internal/db"
)

// newTestDB opens a temp-file SQLite via the production db.Open path
// (so it runs the real migrations and uses the same PRAGMAs). Each
// test gets its own isolated file, cleaned up by t.TempDir.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Config{
		DatabaseURL:   filepath.Join(dir, "test.db"),
		SessionSecret: "test", // bypasses Load's required-var checks; not used here.
		ESVAPIKey:     "test",
	}
	d, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
