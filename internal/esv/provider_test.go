package esv

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"study-help/internal/scripture"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newProviderWithStub(stub roundTripFunc) *Provider {
	c := NewClient("test-key")
	HTTPClientForTest(c).Transport = stub
	return NewProviderFromClient(c)
}

func stubResponse(status int, body string) roundTripFunc {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}
}

func TestProviderFetchMaps429ToScriptureRateLimited(t *testing.T) {
	p := newProviderWithStub(stubResponse(http.StatusTooManyRequests, ""))
	_, err := p.Fetch(context.Background(), "John 1:1", scripture.Options{})
	if !errors.Is(err, scripture.ErrRateLimited) {
		t.Fatalf("err = %v, want scripture.ErrRateLimited", err)
	}
}

func TestProviderFetchMaps5xxToScriptureUpstream(t *testing.T) {
	p := newProviderWithStub(stubResponse(http.StatusBadGateway, ""))
	_, err := p.Fetch(context.Background(), "John 1:1", scripture.Options{})
	if !errors.Is(err, scripture.ErrUpstream) {
		t.Fatalf("err = %v, want scripture.ErrUpstream", err)
	}
}

func TestProviderFetchMapsTransportFailureToScriptureUpstream(t *testing.T) {
	stub := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: connection refused")
	})
	p := newProviderWithStub(stub)
	_, err := p.Fetch(context.Background(), "John 1:1", scripture.Options{})
	if !errors.Is(err, scripture.ErrUpstream) {
		t.Fatalf("err = %v, want scripture.ErrUpstream", err)
	}
}

func TestProviderFetchPassesThroughOnSuccess(t *testing.T) {
	const body = `{"passages":["In the beginning..."]}`
	stub := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})
	p := newProviderWithStub(stub)
	res, err := p.Fetch(context.Background(), "John 1:1", scripture.Options{IncludeHeadings: true})
	if err != nil {
		t.Fatalf("Fetch err = %v, want nil", err)
	}
	if string(res.Body) != body {
		t.Errorf("Body = %q, want %q", res.Body, body)
	}
	if res.Status != http.StatusOK {
		t.Errorf("Status = %d, want 200", res.Status)
	}
	if res.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want application/json", res.ContentType)
	}
}

func TestProviderIDAndDisplayName(t *testing.T) {
	p := NewProvider("ignored")
	if p.ID() != scripture.ESV {
		t.Errorf("ID() = %q, want %q", p.ID(), scripture.ESV)
	}
	if p.DisplayName() == "" {
		t.Error("DisplayName() empty")
	}
}
