package auth

import "testing"

func TestPerIPLimiterAllowsBurstThenRejects(t *testing.T) {
	l := NewLimiter()
	addr := "1.2.3.4:1111"
	for i := 0; i < rlIPBurst; i++ {
		if !l.AllowIP(addr) {
			t.Fatalf("attempt %d: rejected within burst", i+1)
		}
	}
	if l.AllowIP(addr) {
		t.Fatal("burst+1 attempt allowed; should be rejected")
	}
}

func TestPerIPLimiterIsolatesSourceIPs(t *testing.T) {
	l := NewLimiter()
	for i := 0; i < rlIPBurst; i++ {
		l.AllowIP("1.2.3.4:1111")
	}
	if !l.AllowIP("9.9.9.9:2222") {
		t.Fatal("a different IP should still be allowed")
	}
}

func TestPerIPLimiterIgnoresPort(t *testing.T) {
	l := NewLimiter()
	for i := 0; i < rlIPBurst; i++ {
		if !l.AllowIP("1.2.3.4:1111") {
			t.Fatalf("attempt %d: rejected", i+1)
		}
	}
	// Same host, different port — should share the bucket.
	if l.AllowIP("1.2.3.4:9999") {
		t.Fatal("different port allowed; should share the host's bucket")
	}
}

func TestPerAccountLimiterTripsAfterFailures(t *testing.T) {
	l := NewLimiter()
	email := "a@b.c"
	for i := 0; i < rlAccountBurst; i++ {
		if !l.AllowAccount(email) {
			t.Fatalf("AllowAccount %d: rejected within burst", i+1)
		}
		l.RecordAccountFailure(email)
	}
	if l.AllowAccount(email) {
		t.Fatal("AllowAccount after burst failures: allowed; should be rejected")
	}
}

func TestPerAccountLimiterNormalizesEmail(t *testing.T) {
	l := NewLimiter()
	for i := 0; i < rlAccountBurst; i++ {
		l.RecordAccountFailure("a@b.c")
	}
	if l.AllowAccount("A@B.C ") {
		t.Fatal("normalized email lookup did not match; should share bucket")
	}
}

func TestSplitHostPortIPv6(t *testing.T) {
	if got := hostOf("[::1]:1234"); got != "::1" {
		t.Errorf("IPv6: got %q, want ::1", got)
	}
	if got := hostOf("1.2.3.4:5678"); got != "1.2.3.4" {
		t.Errorf("IPv4: got %q, want 1.2.3.4", got)
	}
	if got := hostOf("noport"); got != "noport" {
		t.Errorf("fallback: got %q, want noport", got)
	}
}
