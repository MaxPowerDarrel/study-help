package highlights

import (
	"context"
	"database/sql"
	"time"
)

type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func insertHighlight(ctx context.Context, tx dbtx, userID int64, h Highlight, now time.Time) (Highlight, error) {
	const q = `INSERT INTO highlights (user_id, book, chapter, start_verse, start_offset, end_verse, end_offset, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, q, userID, h.Book, h.Chapter, h.StartVerse, h.StartOffset, h.EndVerse, h.EndOffset, now)
	if err != nil {
		return Highlight{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Highlight{}, err
	}
	h.ID = id
	h.CreatedAt = now
	return h, nil
}

func listHighlights(ctx context.Context, tx dbtx, userID int64, book string, chapter int) ([]Highlight, error) {
	const q = `SELECT id, book, chapter, start_verse, start_offset, end_verse, end_offset, created_at
	FROM highlights
	WHERE user_id = ? AND book = ? AND chapter = ?
	ORDER BY start_verse, start_offset`
	rows, err := tx.QueryContext(ctx, q, userID, book, chapter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Highlight{}
	for rows.Next() {
		var h Highlight
		if err := rows.Scan(&h.ID, &h.Book, &h.Chapter, &h.StartVerse, &h.StartOffset, &h.EndVerse, &h.EndOffset, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// deleteHighlight removes the row only when (id, user_id) matches.
// A non-matching row (different user, or no such id) returns ErrNotFound.
// This collapses the 403/404 distinction the spec calls for.
func deleteHighlight(ctx context.Context, tx dbtx, userID, id int64) error {
	const q = `DELETE FROM highlights WHERE id = ? AND user_id = ?`
	res, err := tx.ExecContext(ctx, q, id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
