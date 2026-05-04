package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// User is the public account record. password_hash is intentionally not
// exposed.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ErrEmailTaken is returned by signup when the normalized email already
// exists in the users table.
var ErrEmailTaken = errors.New("auth: email already in use")

// ErrUserNotFound is returned when a lookup by ID or email yields no row.
var ErrUserNotFound = errors.New("auth: user not found")

// normalizeEmail lowercases and trims whitespace. Per
// specs/accounts.md decision: simplest correct rule, +-aliases distinct.
func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func insertUser(ctx context.Context, tx dbtx, email string, passwordHash []byte, now time.Time) (*User, error) {
	const q = `INSERT INTO users (email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, q, email, string(passwordHash), now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Email: email, CreatedAt: now, UpdatedAt: now}, nil
}

func findUserByEmail(ctx context.Context, tx dbtx, email string) (*User, []byte, error) {
	const q = `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ?`
	var u User
	var hash string
	err := tx.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Email, &hash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrUserNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	return &u, []byte(hash), nil
}

func findUserByID(ctx context.Context, tx dbtx, id int64) (*User, error) {
	const q = `SELECT id, email, created_at, updated_at FROM users WHERE id = ?`
	var u User
	err := tx.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// isUniqueViolation detects the modernc/sqlite "UNIQUE constraint failed"
// error. The driver returns errors with that substring; matching on it
// keeps us off driver-specific error codes.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
