package notes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"study-help/internal/auth"
)

func authedReq(method, target string, body []byte, token string) *http.Request {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
	}
	if token != "" {
		r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	}
	r.RemoteAddr = "1.2.3.4:1111"
	return r
}

func callViaMiddleware(authSvc *auth.Service, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	authSvc.Middleware(h).ServeHTTP(w, req)
	return w
}

// routedMux returns a ServeMux wired exactly like the real server so
// path-parameter routes (PATCH/DELETE /api/notes/{id}) are exercised.
func routedMux(nSvc *Service) *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/notes", nSvc.HandleList())
	m.HandleFunc("POST /api/notes", nSvc.HandleCreate())
	m.HandleFunc("PATCH /api/notes/{id}", nSvc.HandlePatch())
	m.HandleFunc("DELETE /api/notes/{id}", nSvc.HandleDelete())
	return m
}

func createBodyJSON(t *testing.T, book string, chapter, sv, so, ev, eo int, body string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"book":         book,
		"chapter":      chapter,
		"start_verse":  sv,
		"start_offset": so,
		"end_verse":    ev,
		"end_offset":   eo,
		"body":         body,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func patchBodyJSON(t *testing.T, body string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"body": body})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestUnauthenticatedRequestsReturn401(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)

	cases := []struct {
		name   string
		method string
		target string
		body   []byte
	}{
		{"list", http.MethodGet, "/api/notes?book=John&chapter=3", nil},
		{"create", http.MethodPost, "/api/notes", createBodyJSON(t, "John", 3, 16, 0, 16, 10, "hi")},
		{"patch", http.MethodPatch, "/api/notes/1", patchBodyJSON(t, "hi")},
		{"delete", http.MethodDelete, "/api/notes/1", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := callViaMiddleware(authSvc, mux, authedReq(tc.method, tc.target, tc.body, ""))
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status %d, want 401; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateAndListRoundTripViaHTTP(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30, "for God so loved"), tok))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create status %d, want 201; body=%s", createW.Code, createW.Body.String())
	}
	var saved Note
	if err := json.Unmarshal(createW.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if saved.ID == 0 {
		t.Errorf("create: missing id; got %+v", saved)
	}
	if saved.Book != "John" {
		t.Errorf("book normalized? got %q, want John", saved.Book)
	}
	if saved.Body != "for God so loved" {
		t.Errorf("body roundtrip: got %q", saved.Body)
	}
	if saved.CreatedAt.IsZero() || saved.UpdatedAt.IsZero() {
		t.Errorf("timestamps not set in response: %+v", saved)
	}
	if !saved.CreatedAt.Equal(saved.UpdatedAt) {
		t.Errorf("fresh note: updated_at != created_at: %+v", saved)
	}

	listW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/notes?book=john&chapter=3", nil, tok))
	if listW.Code != http.StatusOK {
		t.Fatalf("list status %d, want 200; body=%s", listW.Code, listW.Body.String())
	}
	var got []Note
	if err := json.Unmarshal(listW.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(got) != 1 || got[0].ID != saved.ID {
		t.Errorf("list mismatch: got %+v", got)
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	cases := []struct {
		name string
		body []byte
	}{
		{"unknown book", createBodyJSON(t, "Hezekiah", 3, 16, 0, 16, 10, "hi")},
		{"chapter too high", createBodyJSON(t, "John", 99, 1, 0, 1, 10, "hi")},
		{"chapter zero", createBodyJSON(t, "John", 0, 1, 0, 1, 10, "hi")},
		{"verse zero", createBodyJSON(t, "John", 3, 0, 0, 1, 10, "hi")},
		{"end before start verse", createBodyJSON(t, "John", 3, 5, 0, 4, 10, "hi")},
		{"empty range same verse", createBodyJSON(t, "John", 3, 1, 5, 1, 5, "hi")},
		{"negative offset", createBodyJSON(t, "John", 3, 1, -1, 1, 5, "hi")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := callViaMiddleware(authSvc, mux,
				authedReq(http.MethodPost, "/api/notes", tc.body, tok))
			if w.Code != http.StatusBadRequest {
				t.Errorf("status %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateAllowsOverlap(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	// Two fully overlapping ranges with different bodies — both succeed.
	w1 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30, "first take"), tok))
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create: status %d body=%s", w1.Code, w1.Body.String())
	}
	w2 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 5, 16, 25, "second take"), tok))
	if w2.Code != http.StatusCreated {
		t.Errorf("overlapping create: status %d, want 201; body=%s", w2.Code, w2.Body.String())
	}
}

func TestCreateRejectsOversizeBody(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	tooBig := strings.Repeat("a", MaxBodyBytes+1)
	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 10, tooBig), tok))
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversize body: status %d, want 400; body=%s", w.Code, w.Body.String())
	}

	// Exactly at cap succeeds.
	atCap := strings.Repeat("a", MaxBodyBytes)
	w2 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 10, atCap), tok))
	if w2.Code != http.StatusCreated {
		t.Errorf("at-cap body: status %d, want 201; body=%s", w2.Code, w2.Body.String())
	}
}

func TestPatchUpdatesBodyAndBumpsUpdatedAt(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 10, "first"), tok))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create: %d", createW.Code)
	}
	var created Note
	_ = json.Unmarshal(createW.Body.Bytes(), &created)

	// Bump the service clock past creation so UpdatedAt is strictly later.
	nSvc.clock = func() time.Time { return created.CreatedAt.Add(5 * time.Minute) }

	patchW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPatch, fmt.Sprintf("/api/notes/%d", created.ID),
			patchBodyJSON(t, "second"), tok))
	if patchW.Code != http.StatusOK {
		t.Fatalf("patch: status %d body=%s", patchW.Code, patchW.Body.String())
	}
	var updated Note
	if err := json.Unmarshal(patchW.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if updated.Body != "second" {
		t.Errorf("body not updated: got %q", updated.Body)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Errorf("updated_at not advanced: %+v", updated)
	}
}

func TestPatchCrossUserReturns404(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, aliceTok := signupUser(t, authSvc, "alice@example.com")
	_, bobTok := signupUser(t, authSvc, "bob@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 10, "alice"), aliceTok))
	var saved Note
	_ = json.Unmarshal(createW.Body.Bytes(), &saved)

	patchW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPatch, fmt.Sprintf("/api/notes/%d", saved.ID),
			patchBodyJSON(t, "bob was here"), bobTok))
	if patchW.Code != http.StatusNotFound {
		t.Errorf("cross-user patch: status %d, want 404; body=%s", patchW.Code, patchW.Body.String())
	}
}

func TestPatchRejectsOversizeBody(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 10, "first"), tok))
	var saved Note
	_ = json.Unmarshal(createW.Body.Bytes(), &saved)

	tooBig := strings.Repeat("a", MaxBodyBytes+1)
	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPatch, fmt.Sprintf("/api/notes/%d", saved.ID),
			patchBodyJSON(t, tooBig), tok))
	if w.Code != http.StatusBadRequest {
		t.Errorf("oversize patch: status %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteCrossUserReturns404(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, aliceTok := signupUser(t, authSvc, "alice@example.com")
	_, bobTok := signupUser(t, authSvc, "bob@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30, "alice"), aliceTok))
	if createW.Code != http.StatusCreated {
		t.Fatalf("alice create: %d", createW.Code)
	}
	var saved Note
	_ = json.Unmarshal(createW.Body.Bytes(), &saved)

	delW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodDelete, fmt.Sprintf("/api/notes/%d", saved.ID), nil, bobTok))
	if delW.Code != http.StatusNotFound {
		t.Errorf("cross-user delete: status %d, want 404; body=%s", delW.Code, delW.Body.String())
	}

	listW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/notes?book=John&chapter=3", nil, aliceTok))
	var got []Note
	_ = json.Unmarshal(listW.Body.Bytes(), &got)
	if len(got) != 1 {
		t.Errorf("alice's note gone after cross-user delete attempt: %+v", got)
	}
}

func TestDeleteOwnedReturns204(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30, "x"), tok))
	var saved Note
	_ = json.Unmarshal(createW.Body.Bytes(), &saved)

	delW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodDelete, fmt.Sprintf("/api/notes/%d", saved.ID), nil, tok))
	if delW.Code != http.StatusNoContent {
		t.Errorf("owner delete: status %d, want 204; body=%s", delW.Code, delW.Body.String())
	}
}

func TestListAcceptsBookAliases(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/notes",
			createBodyJSON(t, "Psalm", 23, 1, 0, 1, 10, "shepherd"), tok))
	if w.Code != http.StatusCreated {
		t.Fatalf("Psalm alias create: status %d body=%s", w.Code, w.Body.String())
	}
	var saved Note
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	if saved.Book != "Psalms" {
		t.Errorf("alias not normalized: got %q, want Psalms", saved.Book)
	}
}

func TestListReturnsEmptyArrayNotNull(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/notes?book=John&chapter=3", nil, tok))
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Errorf("empty list body = %q, want \"[]\" (not null)", body)
	}
}

func TestListOrderedByAnchor(t *testing.T) {
	nSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(nSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	// Insert out-of-order: v17, v16(off=30), v16(off=0).
	for _, body := range [][]byte{
		createBodyJSON(t, "John", 3, 17, 5, 17, 20, "third"),
		createBodyJSON(t, "John", 3, 16, 30, 16, 50, "second"),
		createBodyJSON(t, "John", 3, 16, 0, 16, 20, "first"),
	} {
		w := callViaMiddleware(authSvc, mux, authedReq(http.MethodPost, "/api/notes", body, tok))
		if w.Code != http.StatusCreated {
			t.Fatalf("create: status %d body=%s", w.Code, w.Body.String())
		}
	}

	listW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/notes?book=John&chapter=3", nil, tok))
	var got []Note
	_ = json.Unmarshal(listW.Body.Bytes(), &got)
	if len(got) != 3 {
		t.Fatalf("want 3 notes, got %d: %+v", len(got), got)
	}
	if got[0].StartVerse != 16 || got[0].StartOffset != 0 ||
		got[1].StartVerse != 16 || got[1].StartOffset != 30 ||
		got[2].StartVerse != 17 {
		t.Errorf("rows not ordered by (start_verse, start_offset): %+v", got)
	}
}
