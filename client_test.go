package tradestation

import (
	"net/http"
	"testing"
)

func TestNewClient_DefaultsToTestEnvironment(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	if c.apiBase != "https://sim-api.tradestation.com" {
		t.Errorf("apiBase = %q, want sim-api", c.apiBase)
	}
}

func TestNewClient_ProductionEnvironment(t *testing.T) {
	c := NewClient(Production, "id", "secret", "refresh")
	if c.apiBase != "https://api.tradestation.com" {
		t.Errorf("apiBase = %q, want api", c.apiBase)
	}
}

func TestNewClient_StoresCredentials(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	if c.clientID != "id" || c.clientSecret != "secret" || c.refreshToken != "refresh" {
		t.Errorf("credentials not stored: got %q/%q/%q", c.clientID, c.clientSecret, c.refreshToken)
	}
}

func TestWithRefreshTokenRotate_StoresCallback(t *testing.T) {
	called := false
	c := NewClient(Test, "id", "secret", "refresh",
		WithRefreshTokenRotate(func(string) { called = true }))
	if c.onRotate == nil {
		t.Fatal("onRotate not set")
	}
	c.onRotate("new")
	if !called {
		t.Error("callback not invoked")
	}
}

func TestWithHTTPClient_PreservesCustomTransport(t *testing.T) {
	custom := &http.Client{Transport: &fakeRoundTripper{}}
	c := NewClient(Test, "id", "secret", "refresh", WithHTTPClient(custom))
	// Our transport wraps theirs — unwrap once and assert.
	at, ok := c.http.Transport.(*authTransport)
	if !ok {
		t.Fatalf("Transport is %T, want *authTransport", c.http.Transport)
	}
	if _, ok := at.base.(*fakeRoundTripper); !ok {
		t.Errorf("base transport is %T, want *fakeRoundTripper", at.base)
	}
}

type fakeRoundTripper struct{}

func (f *fakeRoundTripper) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }
