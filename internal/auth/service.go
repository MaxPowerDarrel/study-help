package auth

import (
	"context"
	"database/sql"
	"errors"
	"study-help/internal/db"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Config holds the knobs the auth Service needs at construction.
type Config struct {
	IsDev      bool
	SessionTTL time.Duration    // default 30 days
	BcryptCost int              // default bcrypt.DefaultCost
	Clock      func() time.Time // default time.Now
}

// Service is the authentication orchestrator: signup, signin, signout,
// session lookup. Pass *sql.DB; transactions are managed internally.
type Service struct {
	db   *sql.DB
	cfg  Config
	now  func() time.Time
	cost int
	ttl  time.Duration
}

// NewService constructs a Service. Zero-valued Config fields get sane
// defaults: 30-day TTL, bcrypt.DefaultCost, time.Now clock.
func NewService(db *sql.DB, cfg Config) *Service {
	s := &Service{db: db, cfg: cfg}
	s.cost = cfg.BcryptCost
	if s.cost == 0 {
		s.cost = bcrypt.DefaultCost
	}
	s.ttl = cfg.SessionTTL
	if s.ttl == 0 {
		s.ttl = 30 * 24 * time.Hour
	}
	s.now = cfg.Clock
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

// IsDev reports whether the service is configured for dev mode. Used by
// HTTP handlers when calling into the cookie helper.
func (s *Service) IsDev() bool { return s.cfg.IsDev }

// Signup creates a new user and an initial session in a single
// transaction. Returns the user and the freshly issued session token.
func (s *Service) Signup(ctx context.Context, email, password string) (*User, sessionToken, error) {
	email = normalizeEmail(email)
	hash, err := hashPassword(password, s.cost)
	if err != nil {
		return nil, sessionToken{}, err
	}
	now := s.now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, sessionToken{}, err
	}
	defer tx.Rollback()

	user, err := insertUser(ctx, tx, email, hash, now)
	if err != nil {
		return nil, sessionToken{}, err
	}
	tok, err := s.issueSession(ctx, tx, user.ID, now)
	if err != nil {
		return nil, sessionToken{}, err
	}
	if err := tx.Commit(); err != nil {
		return nil, sessionToken{}, err
	}
	return user, tok, nil
}

// Signin validates credentials and issues a fresh session. Returns
// ErrInvalidCredentials for both unknown email and wrong password
// (handler-side message stays ambiguous to avoid enumeration).
func (s *Service) Signin(ctx context.Context, email, password string) (*User, sessionToken, error) {
	email = normalizeEmail(email)

	user, hash, err := findUserByEmail(ctx, s.db, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, sessionToken{}, ErrInvalidCredentials
		}
		return nil, sessionToken{}, err
	}
	if err := verifyPassword(hash, password); err != nil {
		return nil, sessionToken{}, err
	}
	now := s.now().UTC()
	tok, err := s.issueSession(ctx, s.db, user.ID, now)
	if err != nil {
		return nil, sessionToken{}, err
	}
	return user, tok, nil
}

// Signout deletes the session row matching rawToken. Idempotent: a
// missing or unparseable token is not an error (the cookie may have
// been tampered with or already expired).
func (s *Service) Signout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	hash, err := hashToken(rawToken)
	if err != nil {
		return nil
	}
	return deleteSessionByHash(ctx, s.db, hash)
}

// Authenticate looks up a session by raw token, bumps the sliding
// window, and returns the user. The returned sessionToken carries the
// refreshed ExpiresAt (Raw is empty — the cookie value itself doesn't
// rotate, only the expiry does).
func (s *Service) Authenticate(ctx context.Context, rawToken string) (*User, sessionToken, error) {
	if rawToken == "" {
		return nil, sessionToken{}, ErrSessionNotFound
	}
	hash, err := hashToken(rawToken)
	if err != nil {
		return nil, sessionToken{}, ErrSessionNotFound
	}
	sess, err := findSessionByHash(ctx, s.db, hash)
	if err != nil {
		return nil, sessionToken{}, err
	}
	now := s.now().UTC()
	if !sess.ExpiresAt.After(now) {
		// Best-effort cleanup; ignore error.
		_ = deleteSession(ctx, s.db, sess.ID)
		return nil, sessionToken{}, ErrSessionExpired
	}
	user, err := findUserByID(ctx, s.db, sess.UserID)
	if err != nil {
		return nil, sessionToken{}, err
	}
	newExpiry := now.Add(s.ttl)
	if err := bumpSession(ctx, s.db, sess.ID, now, newExpiry); err != nil {
		return nil, sessionToken{}, err
	}
	return user, sessionToken{Raw: rawToken, ExpiresAt: newExpiry}, nil
}

func (s *Service) issueSession(ctx context.Context, tx db.TX, userID int64, now time.Time) (sessionToken, error) {
	raw, hash, err := newSessionToken()
	if err != nil {
		return sessionToken{}, err
	}
	expiresAt := now.Add(s.ttl)
	if _, err := insertSession(ctx, tx, userID, hash, now, expiresAt); err != nil {
		return sessionToken{}, err
	}
	return sessionToken{Raw: raw, ExpiresAt: expiresAt}, nil
}
