package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"
)

// SessionTokenBytes is the entropy size for session cookie values.
// 32 bytes = 256 bits, encoded base64url (43 chars, no padding).
const SessionTokenBytes = 32

// session is the internal session row. Not exported — callers use
// sessionToken to know what to write back into the cookie.
type session struct {
	ID         int64
	UserID     int64
	CreatedAt  time.Time
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

// sessionToken bundles the raw cookie value with its expiry so callers
// can re-issue Set-Cookie with a matching Max-Age. Raw is empty when
// the caller doesn't need to set a cookie (e.g. middleware refresh
// where the cookie value didn't rotate).
type sessionToken struct {
	Raw       string
	ExpiresAt time.Time
}

var (
	// ErrSessionNotFound is returned when a token doesn't match any row.
	ErrSessionNotFound = errors.New("auth: session not found")
	// ErrSessionExpired is returned when the matched row is past expires_at.
	ErrSessionExpired = errors.New("auth: session expired")
)

// newSessionToken generates a fresh token: random bytes from crypto/rand,
// returned as a base64url string for the cookie value, with sha256 of
// the raw bytes for storage. Hashing the bytes (not the encoded string)
// keeps storage independent of any future encoding change.
func newSessionToken() (raw string, hash []byte, err error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(b), sum[:], nil
}

// hashToken returns sha256(raw_bytes). The input is the base64url cookie
// value; we decode it back to bytes before hashing so writes and reads
// see the same input.
func hashToken(raw string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	return sum[:], nil
}

func insertSession(ctx context.Context, tx dbtx, userID int64, hash []byte, now, expiresAt time.Time) (*session, error) {
	const q = `INSERT INTO sessions (user_id, token_hash, created_at, last_seen_at, expires_at) VALUES (?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, q, userID, hash, now, now, expiresAt)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &session{ID: id, UserID: userID, CreatedAt: now, LastSeenAt: now, ExpiresAt: expiresAt}, nil
}

func findSessionByHash(ctx context.Context, tx dbtx, hash []byte) (*session, error) {
	const q = `SELECT id, user_id, created_at, last_seen_at, expires_at FROM sessions WHERE token_hash = ?`
	var s session
	err := tx.QueryRowContext(ctx, q, hash).Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func bumpSession(ctx context.Context, tx dbtx, sessionID int64, now, expiresAt time.Time) error {
	const q = `UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?`
	_, err := tx.ExecContext(ctx, q, now, expiresAt, sessionID)
	return err
}

func deleteSession(ctx context.Context, tx dbtx, sessionID int64) error {
	const q = `DELETE FROM sessions WHERE id = ?`
	_, err := tx.ExecContext(ctx, q, sessionID)
	return err
}

func deleteSessionByHash(ctx context.Context, tx dbtx, hash []byte) error {
	const q = `DELETE FROM sessions WHERE token_hash = ?`
	_, err := tx.ExecContext(ctx, q, hash)
	return err
}
