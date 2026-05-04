package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerifyRoundtrip(t *testing.T) {
	pw := "correcthorsebatterystaple"
	hash, err := hashPassword(pw, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if err := verifyPassword(hash, pw); err != nil {
		t.Fatalf("verifyPassword: %v", err)
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	hash, err := hashPassword("twelvecharspw", bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if err := verifyPassword(hash, "wrongpassword"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("verifyPassword wrong: got %v, want ErrInvalidCredentials", err)
	}
}

func TestHashRejectsShortPassword(t *testing.T) {
	short := strings.Repeat("a", MinPasswordLen-1)
	if _, err := hashPassword(short, bcrypt.MinCost); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("hashPassword short: got %v, want ErrWeakPassword", err)
	}
}

func TestHashAcceptsExactlyMinLen(t *testing.T) {
	pw := strings.Repeat("a", MinPasswordLen)
	if _, err := hashPassword(pw, bcrypt.MinCost); err != nil {
		t.Fatalf("hashPassword min-len: %v", err)
	}
}
