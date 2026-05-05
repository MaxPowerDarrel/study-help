package notes

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInsertAndListRoundTrip(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	user, _ := signupUser(t, authSvc, "alice@example.com")
	ctx := context.Background()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)

	n1 := Note{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 30, Body: "for God so loved"}
	n2 := Note{Book: "John", Chapter: 3, StartVerse: 17, StartOffset: 5, EndVerse: 17, EndOffset: 20, Body: "context for v16"}
	n3 := Note{Book: "John", Chapter: 4, StartVerse: 1, StartOffset: 0, EndVerse: 1, EndOffset: 10, Body: "different chapter"}

	for _, n := range []Note{n1, n2, n3} {
		if _, err := insertNote(ctx, nSvc.db, user.ID, n, now); err != nil {
			t.Fatalf("insertNote: %v", err)
		}
	}

	got, err := listNotes(ctx, nSvc.db, user.ID, "John", 3)
	if err != nil {
		t.Fatalf("listNotes: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 notes for John 3, got %d: %+v", len(got), got)
	}
	if got[0].StartVerse != 16 || got[1].StartVerse != 17 {
		t.Errorf("rows out of order: %+v", got)
	}
	for _, n := range got {
		if n.ID == 0 {
			t.Errorf("note missing id: %+v", n)
		}
		if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
			t.Errorf("note missing timestamps: %+v", n)
		}
		if !n.CreatedAt.Equal(n.UpdatedAt) {
			t.Errorf("fresh note has updated_at != created_at: %+v", n)
		}
	}

	other, _ := signupUser(t, authSvc, "bob@example.com")
	otherList, err := listNotes(ctx, nSvc.db, other.ID, "John", 3)
	if err != nil {
		t.Fatalf("listNotes for other user: %v", err)
	}
	if len(otherList) != 0 {
		t.Errorf("other user saw %d notes; want 0", len(otherList))
	}
}

func TestUpdateBodyBumpsUpdatedAt(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	user, _ := signupUser(t, authSvc, "alice@example.com")
	ctx := context.Background()
	created := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	edited := created.Add(5 * time.Minute)

	saved, err := insertNote(ctx, nSvc.db, user.ID,
		Note{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 10, Body: "first"}, created)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	updated, err := updateNoteBody(ctx, nSvc.db, user.ID, saved.ID, "second", edited)
	if err != nil {
		t.Fatalf("updateNoteBody: %v", err)
	}
	if updated.Body != "second" {
		t.Errorf("body not updated: got %q", updated.Body)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Errorf("updated_at not advanced: created=%v updated=%v", updated.CreatedAt, updated.UpdatedAt)
	}
	if !updated.CreatedAt.Equal(saved.CreatedAt) {
		t.Errorf("created_at changed by update: was %v now %v", saved.CreatedAt, updated.CreatedAt)
	}
}

func TestUpdateAndGetCrossUserReturnNotFound(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	alice, _ := signupUser(t, authSvc, "alice@example.com")
	bob, _ := signupUser(t, authSvc, "bob@example.com")
	ctx := context.Background()
	now := time.Now()

	saved, err := insertNote(ctx, nSvc.db, alice.ID,
		Note{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 10, Body: "alice"}, now)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if _, err := getNote(ctx, nSvc.db, bob.ID, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-user get: err=%v, want ErrNotFound", err)
	}
	if _, err := updateNoteBody(ctx, nSvc.db, bob.ID, saved.ID, "bob's edit", now); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-user update: err=%v, want ErrNotFound", err)
	}

	stillAlice, _ := getNote(ctx, nSvc.db, alice.ID, saved.ID)
	if stillAlice.Body != "alice" {
		t.Errorf("body mutated by cross-user attempt: %q", stillAlice.Body)
	}
}

func TestDeleteOnlyOwnedRow(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	alice, _ := signupUser(t, authSvc, "alice@example.com")
	bob, _ := signupUser(t, authSvc, "bob@example.com")
	ctx := context.Background()
	now := time.Now()

	saved, err := insertNote(ctx, nSvc.db, alice.ID,
		Note{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 10, Body: "x"}, now)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := deleteNote(ctx, nSvc.db, bob.ID, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-user delete: err=%v, want ErrNotFound", err)
	}
	stillThere, _ := listNotes(ctx, nSvc.db, alice.ID, "John", 3)
	if len(stillThere) != 1 {
		t.Errorf("cross-user delete affected the row: alice now has %d notes", len(stillThere))
	}

	if err := deleteNote(ctx, nSvc.db, alice.ID, saved.ID); err != nil {
		t.Errorf("owner delete: %v", err)
	}
	gone, _ := listNotes(ctx, nSvc.db, alice.ID, "John", 3)
	if len(gone) != 0 {
		t.Errorf("owner delete didn't remove row")
	}

	if err := deleteNote(ctx, nSvc.db, alice.ID, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("repeat delete: err=%v, want ErrNotFound", err)
	}
}

func TestUserDeletionCascades(t *testing.T) {
	nSvc, authSvc, d := newTestStack(t)
	alice, _ := signupUser(t, authSvc, "alice@example.com")
	ctx := context.Background()

	if _, err := insertNote(ctx, nSvc.db, alice.ID,
		Note{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 10, Body: "x"}, time.Now()); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if _, err := d.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, alice.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var n int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes WHERE user_id = ?`, alice.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("ON DELETE CASCADE didn't fire: %d rows remain", n)
	}
}
