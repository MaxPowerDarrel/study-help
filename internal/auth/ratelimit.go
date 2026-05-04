package auth

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Rate-limit knobs. Per specs/accounts.md decision: ~10 attempts / 15min
// per IP on signup+signin, plus ~5 failures / 15min per account on signin.
const (
	rlWindow              = 15 * time.Minute
	rlIPBurst             = 10
	rlAccountBurst        = 5
	rlCleanupThreshold    = 10_000
	rlCleanupAgeMultipler = 2 // entries older than this many windows are pruned
)

// Limiter is a dual-bucket rate limiter: per-source-IP and per-account.
// In-memory only; resets on process restart (acceptable per spec — a
// deploy effectively whitelists everyone for one window).
//
// TODO(accounts): add support for a trusted X-Forwarded-For header
// when deployed behind a reverse proxy.
type Limiter struct {
	mu sync.Mutex

	ipBuckets   map[string]*rateEntry
	failBuckets map[string]*rateEntry

	ipLimit      rate.Limit
	accountLimit rate.Limit
}

type rateEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// NewLimiter constructs a Limiter with the spec-defined defaults.
func NewLimiter() *Limiter {
	return &Limiter{
		ipBuckets:    make(map[string]*rateEntry),
		failBuckets:  make(map[string]*rateEntry),
		ipLimit:      rate.Every(rlWindow / rlIPBurst),
		accountLimit: rate.Every(rlWindow / rlAccountBurst),
	}
}

// AllowIP consumes one IP-bucket token for the given remoteAddr. The
// host portion is used as the key (TCP source ports differ across
// connections from the same client). Returns false when the bucket is
// empty.
func (l *Limiter) AllowIP(remoteAddr string) bool {
	host := hostOf(remoteAddr)
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.getOrCreate(l.ipBuckets, host, l.ipLimit, rlIPBurst)
	return e.lim.Allow()
}

// AllowAccount peeks at the per-account bucket without consuming a
// token. Use this on signin to fail-fast when the bucket is empty;
// successful signins should not burn account-bucket capacity. Tokens
// are consumed by RecordAccountFailure.
func (l *Limiter) AllowAccount(email string) bool {
	key := normalizeEmail(email)
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.getOrCreate(l.failBuckets, key, l.accountLimit, rlAccountBurst)
	// AllowN(now, 0) returns whether at least one token is available
	// without consuming. (n=0 always allows; instead we check Tokens.)
	return e.lim.Tokens() >= 1
}

// RecordAccountFailure consumes one per-account token after a failed
// signin attempt. Always returns; the caller has already decided to
// 401 the request.
func (l *Limiter) RecordAccountFailure(email string) {
	key := normalizeEmail(email)
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.getOrCreate(l.failBuckets, key, l.accountLimit, rlAccountBurst)
	_ = e.lim.Allow()
}

func (l *Limiter) getOrCreate(m map[string]*rateEntry, key string, limit rate.Limit, burst int) *rateEntry {
	now := time.Now()
	e, ok := m[key]
	if !ok {
		e = &rateEntry{lim: rate.NewLimiter(limit, burst)}
		m[key] = e
	}
	e.lastSeen = now
	if len(m) > rlCleanupThreshold {
		cutoff := now.Add(-rlCleanupAgeMultipler * rlWindow)
		for k, v := range m {
			if v.lastSeen.Before(cutoff) {
				delete(m, k)
			}
		}
	}
	return e
}

// hostOf strips the port from a "host:port" RemoteAddr. Falls back to
// the raw input on parse failure (httptest requests sometimes carry
// no port).
func hostOf(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
