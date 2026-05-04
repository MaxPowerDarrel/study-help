package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLen is the minimum password length enforced at signup.
// Per specs/accounts.md decision (2026-05-04): length-only policy, no
// complexity rules. NIST-aligned guidance favors length over complexity.
const MinPasswordLen = 12

// ErrWeakPassword is returned when a password is shorter than MinPasswordLen.
var ErrWeakPassword = errors.New("auth: password too short")

// ErrInvalidCredentials is returned for any signin failure (wrong password,
// unknown email). The error message at the HTTP layer is deliberately
// ambiguous to avoid user enumeration.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")

func hashPassword(plain string, cost int) ([]byte, error) {
	if len(plain) < MinPasswordLen {
		return nil, ErrWeakPassword
	}
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return bcrypt.GenerateFromPassword([]byte(plain), cost)
}

func verifyPassword(hash []byte, plain string) error {
	if err := bcrypt.CompareHashAndPassword(hash, []byte(plain)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}
