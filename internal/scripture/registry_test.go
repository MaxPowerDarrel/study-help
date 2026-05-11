package scripture

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	id   ID
	name string
}

func (f *fakeProvider) ID() ID              { return f.id }
func (f *fakeProvider) DisplayName() string { return f.name }
func (f *fakeProvider) Fetch(context.Context, string, Options) (*Result, error) {
	return &Result{Body: []byte("ok"), ContentType: "text/plain", Status: 200}, nil
}

func TestRegistryGetUnknown(t *testing.T) {
	r := NewRegistry(ESV, &fakeProvider{id: ESV, name: "ESV"})
	if _, err := r.Get("NIV"); !errors.Is(err, ErrUnknown) {
		t.Fatalf("Get(unknown) = %v, want ErrUnknown", err)
	}
}

func TestRegistryGetKnown(t *testing.T) {
	want := &fakeProvider{id: ESV, name: "ESV"}
	r := NewRegistry(ESV, want)
	got, err := r.Get(ESV)
	if err != nil {
		t.Fatalf("Get(ESV) err=%v", err)
	}
	if got != want {
		t.Fatalf("Get(ESV) returned a different provider")
	}
}

func TestRegistryDefault(t *testing.T) {
	r := NewRegistry(ESV,
		&fakeProvider{id: "KJV", name: "KJV"},
		&fakeProvider{id: ESV, name: "ESV"},
	)
	if r.Default() != ESV {
		t.Fatalf("Default() = %v, want ESV (registration order shouldn't change fallback)", r.Default())
	}
}

func TestRegistryKnown(t *testing.T) {
	r := NewRegistry(ESV, &fakeProvider{id: ESV, name: "ESV"})
	if !r.Known(ESV) {
		t.Errorf("Known(ESV) = false, want true")
	}
	if r.Known("BOGUS") {
		t.Errorf("Known(BOGUS) = true, want false")
	}
}

func TestRegistryDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("registering duplicate IDs did not panic")
		}
	}()
	NewRegistry(ESV,
		&fakeProvider{id: ESV, name: "first"},
		&fakeProvider{id: ESV, name: "second"},
	)
}

func TestRegistryUnregisteredFallbackPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("unregistered fallback did not panic")
		}
	}()
	NewRegistry("KJV", &fakeProvider{id: ESV, name: "ESV"})
}
