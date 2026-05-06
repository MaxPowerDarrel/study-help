package youversion

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"study-help/internal/scripture"
)

func TestProvider_IDAndDisplayName(t *testing.T) {
	p := NewProvider("test")
	if p.ID() != scripture.NIV {
		t.Errorf("ID() = %q, want %q", p.ID(), scripture.NIV)
	}
	if p.DisplayName() != "New International Version" {
		t.Errorf("DisplayName() = %q", p.DisplayName())
	}
}

func TestProvider_RemapsRateLimited(t *testing.T) {
	c := withStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	p := NewProviderFromClient(c)
	_, err := p.Fetch(context.Background(), "John 3", scripture.Options{})
	if !errors.Is(err, scripture.ErrRateLimited) {
		t.Errorf("err = %v, want scripture.ErrRateLimited", err)
	}
}

func TestProvider_RemapsUpstream(t *testing.T) {
	c := withStub(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	p := NewProviderFromClient(c)
	_, err := p.Fetch(context.Background(), "John 3", scripture.Options{})
	if !errors.Is(err, scripture.ErrUpstream) {
		t.Errorf("err = %v, want scripture.ErrUpstream", err)
	}
}
