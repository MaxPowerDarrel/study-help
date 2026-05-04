package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestNewSessionTokenIsRandom(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		raw, _, err := newSessionToken()
		if err != nil {
			t.Fatalf("newSessionToken: %v", err)
		}
		if seen[raw] {
			t.Fatalf("duplicate token after %d iterations: %s", i, raw)
		}
		seen[raw] = true
	}
}

func TestStoredHashDiffersFromRawToken(t *testing.T) {
	raw, hash, err := newSessionToken()
	if err != nil {
		t.Fatalf("newSessionToken: %v", err)
	}
	if bytes.Equal([]byte(raw), hash) {
		t.Fatal("stored hash equals raw token bytes; should be sha256")
	}
	// Decode raw → bytes → sha256 should equal hash (the stored form).
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	want := sha256.Sum256(decoded)
	if !bytes.Equal(hash, want[:]) {
		t.Fatal("hash != sha256(decoded raw)")
	}
}

func TestHashTokenRoundTrip(t *testing.T) {
	raw, hash, err := newSessionToken()
	if err != nil {
		t.Fatalf("newSessionToken: %v", err)
	}
	got, err := hashToken(raw)
	if err != nil {
		t.Fatalf("hashToken: %v", err)
	}
	if !bytes.Equal(got, hash) {
		t.Fatal("hashToken(raw) != hash returned by newSessionToken")
	}
}

func TestHashTokenRejectsGarbage(t *testing.T) {
	if _, err := hashToken("not!valid!base64url"); err == nil {
		t.Fatal("hashToken: want error for invalid base64url, got nil")
	}
}
