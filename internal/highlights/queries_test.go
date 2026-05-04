package highlights

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInsertAndListRoundTrip(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	user, _ := signupUser(t, authSvc, "alice@example.com")
	ctx := context.Background()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)

	h1 := Highlight{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 30}
	h2 := Highlight{Book: "John", Chapter: 3, StartVerse: 17, StartOffset: 5, EndVerse: 17, EndOffset: 20}
	h3 := Highlight{Book: "John", Chapter: 4, StartVerse: 1, StartOffset: 0, EndVerse: 1, EndOffset: 10}

	for _, h := range []Highlight{h1, h2, h3} {
		if _, err := insertHighlight(ctx, hSvc.db, user.ID, h, now); err != nil {
			t.Fatalf("insertHighlight: %v", err)
		}
	}

	got, err := listHighlights(ctx, hSvc.db, user.ID, "John", 3)
	if err != nil {
		t.Fatalf("listHighlights: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 highlights for John 3, got %d: %+v", len(got), got)
	}
	if got[0].StartVerse != 16 || got[1].StartVerse != 17 {
		t.Errorf("rows out of order: %+v", got)
	}
	for _, h := range got {
		if h.ID == 0 {
			t.Errorf("highlight missing id: %+v", h)
		}
		if h.CreatedAt.IsZero() {
			t.Errorf("highlight missing created_at: %+v", h)
		}
	}

	other, _ := signupUser(t, authSvc, "bob@example.com")
	otherList, err := listHighlights(ctx, hSvc.db, other.ID, "John", 3)
	if err != nil {
		t.Fatalf("listHighlights for other user: %v", err)
	}
	if len(otherList) != 0 {
		t.Errorf("other user saw %d highlights; want 0", len(otherList))
	}
}

func TestDeleteOnlyOwnedRow(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	alice, _ := signupUser(t, authSvc, "alice@example.com")
	bob, _ := signupUser(t, authSvc, "bob@example.com")
	ctx := context.Background()
	now := time.Now()

	h := Highlight{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 10}
	saved, err := insertHighlight(ctx, hSvc.db, alice.ID, h, now)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := deleteHighlight(ctx, hSvc.db, bob.ID, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("cross-user delete: err=%v, want ErrNotFound", err)
	}
	stillThere, _ := listHighlights(ctx, hSvc.db, alice.ID, "John", 3)
	if len(stillThere) != 1 {
		t.Errorf("cross-user delete affected the row: alice now has %d highlights", len(stillThere))
	}

	if err := deleteHighlight(ctx, hSvc.db, alice.ID, saved.ID); err != nil {
		t.Errorf("owner delete: %v", err)
	}
	gone, _ := listHighlights(ctx, hSvc.db, alice.ID, "John", 3)
	if len(gone) != 0 {
		t.Errorf("owner delete didn't remove row")
	}

	if err := deleteHighlight(ctx, hSvc.db, alice.ID, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("repeat delete: err=%v, want ErrNotFound", err)
	}
}

func TestUserDeletionCascades(t *testing.T) {
	hSvc, authSvc, d := newTestStack(t)
	alice, _ := signupUser(t, authSvc, "alice@example.com")
	ctx := context.Background()

	h := Highlight{Book: "John", Chapter: 3, StartVerse: 16, StartOffset: 0, EndVerse: 16, EndOffset: 10}
	if _, err := insertHighlight(ctx, hSvc.db, alice.ID, h, time.Now()); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if _, err := d.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, alice.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var n int
	if err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM highlights WHERE user_id = ?`, alice.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("ON DELETE CASCADE didn't fire: %d rows remain", n)
	}
}
