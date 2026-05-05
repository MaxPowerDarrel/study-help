package highlights

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
// path-parameter routes (DELETE /api/highlights/{id}) are exercised.
func routedMux(hSvc *Service) *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/highlights", hSvc.HandleList())
	m.HandleFunc("POST /api/highlights", hSvc.HandleCreate())
	m.HandleFunc("DELETE /api/highlights/{id}", hSvc.HandleDelete())
	return m
}

func createBodyJSON(t *testing.T, book string, chapter, sv, so, ev, eo int) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"book":              book,
		"chapter":           chapter,
		"start_verse":       sv,
		"start_char_offset": so,
		"end_verse":         ev,
		"end_char_offset":   eo,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestUnauthenticatedRequestsReturn401(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)

	cases := []struct {
		name    string
		method  string
		target  string
		body    []byte
		wantErr string
	}{
		{"list", http.MethodGet, "/api/highlights?book=John&chapter=3", nil, "unauthorized"},
		{"create", http.MethodPost, "/api/highlights", createBodyJSON(t, "John", 3, 16, 0, 16, 10), "unauthorized"},
		{"delete", http.MethodDelete, "/api/highlights/1", nil, "unauthorized"},
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
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30), tok))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create status %d, want 201; body=%s", createW.Code, createW.Body.String())
	}
	var saved Highlight
	if err := json.Unmarshal(createW.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if saved.ID == 0 {
		t.Errorf("create: missing id; got %+v", saved)
	}
	if saved.Book != "John" {
		t.Errorf("book normalized? got %q, want John", saved.Book)
	}
	if saved.CreatedAt.IsZero() {
		t.Errorf("created_at not set in response: %+v", saved)
	}

	listW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/highlights?book=john&chapter=3", nil, tok))
	if listW.Code != http.StatusOK {
		t.Fatalf("list status %d, want 200; body=%s", listW.Code, listW.Body.String())
	}
	var got []Highlight
	if err := json.Unmarshal(listW.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(got) != 1 || got[0].ID != saved.ID {
		t.Errorf("list mismatch: got %+v", got)
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	cases := []struct {
		name string
		body []byte
	}{
		{"unknown book", createBodyJSON(t, "Hezekiah", 3, 16, 0, 16, 10)},
		{"chapter too high", createBodyJSON(t, "John", 99, 1, 0, 1, 10)},
		{"chapter zero", createBodyJSON(t, "John", 0, 1, 0, 1, 10)},
		{"verse zero", createBodyJSON(t, "John", 3, 0, 0, 1, 10)},
		{"end before start verse", createBodyJSON(t, "John", 3, 5, 0, 4, 10)},
		{"empty range same verse", createBodyJSON(t, "John", 3, 1, 5, 1, 5)},
		{"negative offset", createBodyJSON(t, "John", 3, 1, -1, 1, 5)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := callViaMiddleware(authSvc, mux,
				authedReq(http.MethodPost, "/api/highlights", tc.body, tok))
			if w.Code != http.StatusBadRequest {
				t.Errorf("status %d, want 400; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateRejectsOverlap(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w1 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30), tok))
	if w1.Code != http.StatusCreated {
		t.Fatalf("first create failed: status=%d body=%s", w1.Code, w1.Body.String())
	}

	w2 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "John", 3, 16, 10, 16, 20), tok))
	if w2.Code != http.StatusConflict {
		t.Errorf("overlap create: status %d, want 409; body=%s", w2.Code, w2.Body.String())
	}

	// Adjacent (boundary touching) is fine — half-open ranges.
	w3 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "John", 3, 16, 30, 16, 40), tok))
	if w3.Code != http.StatusCreated {
		t.Errorf("adjacent create: status %d, want 201; body=%s", w3.Code, w3.Body.String())
	}
}

func TestDeleteCrossUserReturns404(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, aliceTok := signupUser(t, authSvc, "alice@example.com")
	_, bobTok := signupUser(t, authSvc, "bob@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30), aliceTok))
	if createW.Code != http.StatusCreated {
		t.Fatalf("alice create: %d", createW.Code)
	}
	var saved Highlight
	_ = json.Unmarshal(createW.Body.Bytes(), &saved)

	delW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodDelete, fmt.Sprintf("/api/highlights/%d", saved.ID), nil, bobTok))
	if delW.Code != http.StatusNotFound {
		t.Errorf("cross-user delete: status %d, want 404; body=%s", delW.Code, delW.Body.String())
	}

	// Alice's row should still be there.
	listW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/highlights?book=John&chapter=3", nil, aliceTok))
	var got []Highlight
	_ = json.Unmarshal(listW.Body.Bytes(), &got)
	if len(got) != 1 {
		t.Errorf("alice's highlight gone after cross-user delete attempt: %+v", got)
	}
}

func TestDeleteOwnedReturns204(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	createW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "John", 3, 16, 0, 16, 30), tok))
	var saved Highlight
	_ = json.Unmarshal(createW.Body.Bytes(), &saved)

	delW := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodDelete, fmt.Sprintf("/api/highlights/%d", saved.ID), nil, tok))
	if delW.Code != http.StatusNoContent {
		t.Errorf("owner delete: status %d, want 204; body=%s", delW.Code, delW.Body.String())
	}
}

func TestListAcceptsBookAliases(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSON(t, "Psalm", 23, 1, 0, 1, 10), tok))
	if w.Code != http.StatusCreated {
		t.Fatalf("Psalm alias create: status %d body=%s", w.Code, w.Body.String())
	}
	var saved Highlight
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	if saved.Book != "Psalms" {
		t.Errorf("alias not normalized: got %q, want Psalms", saved.Book)
	}
}

func TestListReturnsEmptyArrayNotNull(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/highlights?book=John&chapter=3", nil, tok))
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "[]" {
		t.Errorf("empty list body = %q, want \"[]\" (not null)", body)
	}
}

// createBodyJSONWithTranslation builds a body that explicitly stamps a
// translation, used to cross-check translation scoping.
func createBodyJSONWithTranslation(t *testing.T, translation, book string, chapter, sv, so, ev, eo int) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"translation":       translation,
		"book":              book,
		"chapter":           chapter,
		"start_verse":       sv,
		"start_char_offset": so,
		"end_verse":         ev,
		"end_char_offset":   eo,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestCreateRejectsUnknownTranslation(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSONWithTranslation(t, "BOGUS", "John", 3, 16, 0, 16, 30), tok))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestListRejectsUnknownTranslation(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	w := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/highlights?book=John&chapter=3&translation=BOGUS", nil, tok))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestCrossTranslationIsolation(t *testing.T) {
	hSvc, authSvc, _ := newTestStack(t)
	mux := routedMux(hSvc)
	_, tok := signupUser(t, authSvc, "alice@example.com")

	// Create one highlight in ESV (the schema default for the new account).
	w1 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSONWithTranslation(t, "ESV", "John", 3, 16, 0, 16, 30), tok))
	if w1.Code != http.StatusCreated {
		t.Fatalf("ESV create: %d body=%s", w1.Code, w1.Body.String())
	}

	// Create one highlight in KJV (registered in newTestRegistry).
	w2 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSONWithTranslation(t, "KJV", "John", 3, 17, 0, 17, 20), tok))
	if w2.Code != http.StatusCreated {
		t.Fatalf("KJV create: %d body=%s", w2.Code, w2.Body.String())
	}

	// List ESV → 1 row only.
	esvList := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/highlights?book=John&chapter=3&translation=ESV", nil, tok))
	var got []Highlight
	_ = json.Unmarshal(esvList.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Translation != "ESV" {
		t.Errorf("ESV list: got %+v, want exactly the ESV highlight", got)
	}

	// List KJV → 1 row only.
	kjvList := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodGet, "/api/highlights?book=John&chapter=3&translation=KJV", nil, tok))
	got = nil
	_ = json.Unmarshal(kjvList.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Translation != "KJV" {
		t.Errorf("KJV list: got %+v, want exactly the KJV highlight", got)
	}

	// Same range across translations doesn't trip the overlap check
	// because overlap is scoped per-translation.
	w3 := callViaMiddleware(authSvc, mux,
		authedReq(http.MethodPost, "/api/highlights",
			createBodyJSONWithTranslation(t, "KJV", "John", 3, 16, 0, 16, 30), tok))
	if w3.Code != http.StatusCreated {
		t.Errorf("same range in different translation: status %d, want 201", w3.Code)
	}
}
