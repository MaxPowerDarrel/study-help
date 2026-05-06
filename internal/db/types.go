package db

import (
	"context"
	"database/sql"
)

// TX is the subset of *sql.DB and *sql.Tx used by query helpers in
// other internal packages. Using an interface lets callers run a query
// against either a connection pool or an open transaction.
type TX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
