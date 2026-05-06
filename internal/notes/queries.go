package notes

import (
	"context"
	"database/sql"
	"time"

	"study-help/internal/db"
)

func insertNote(ctx context.Context, tx db.TX, userID int64, n Note, now time.Time) (Note, error) {
	const q = `INSERT INTO notes (user_id, translation, book, chapter, start_verse, start_offset, end_verse, end_offset, body, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, q, userID, n.Translation, n.Book, n.Chapter, n.StartVerse, n.StartOffset, n.EndVerse, n.EndOffset, n.Body, now, now)
	if err != nil {
		return Note{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Note{}, err
	}
	n.ID = id
	n.CreatedAt = now
	n.UpdatedAt = now
	return n, nil
}

func listNotes(ctx context.Context, tx db.TX, userID int64, translation, book string, chapter int) ([]Note, error) {
	const q = `SELECT id, translation, book, chapter, start_verse, start_offset, end_verse, end_offset, body, created_at, updated_at
	FROM notes
	WHERE user_id = ? AND translation = ? AND book = ? AND chapter = ?
	ORDER BY start_verse, start_offset`
	rows, err := tx.QueryContext(ctx, q, userID, translation, book, chapter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Translation, &n.Book, &n.Chapter, &n.StartVerse, &n.StartOffset, &n.EndVerse, &n.EndOffset, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// getNote returns the single note (id, user_id) tuple owns. A miss —
// non-existent id, or row owned by another user — returns ErrNotFound,
// collapsing the 403/404 distinction the same way deleteNote does.
func getNote(ctx context.Context, tx db.TX, userID, id int64) (Note, error) {
	const q = `SELECT id, translation, book, chapter, start_verse, start_offset, end_verse, end_offset, body, created_at, updated_at
	FROM notes
	WHERE id = ? AND user_id = ?`
	row := tx.QueryRowContext(ctx, q, id, userID)
	var n Note
	if err := row.Scan(&n.ID, &n.Translation, &n.Book, &n.Chapter, &n.StartVerse, &n.StartOffset, &n.EndVerse, &n.EndOffset, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return Note{}, ErrNotFound
		}
		return Note{}, err
	}
	return n, nil
}

func updateNoteBody(ctx context.Context, tx db.TX, userID, id int64, body string, now time.Time) (Note, error) {
	const q = `UPDATE notes SET body = ?, updated_at = ? WHERE id = ? AND user_id = ?`
	res, err := tx.ExecContext(ctx, q, body, now, id, userID)
	if err != nil {
		return Note{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return Note{}, err
	}
	if rows == 0 {
		return Note{}, ErrNotFound
	}
	return getNote(ctx, tx, userID, id)
}

// deleteNote removes the row only when (id, user_id) matches.
// A non-matching row (different user, or no such id) returns ErrNotFound.
func deleteNote(ctx context.Context, tx db.TX, userID, id int64) error {
	const q = `DELETE FROM notes WHERE id = ? AND user_id = ?`
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
