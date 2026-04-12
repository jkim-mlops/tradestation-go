package tradestation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestDoJSON_GetHappyPath(t *testing.T) {
	var gotMethod, gotPath, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":"hello"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	var out struct{ Value string }
	if err := c.doJSON(context.Background(), "GET", "/ping", nil, nil, &out); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if gotMethod != "GET" || gotPath != "/ping" || gotAccept != "application/json" {
		t.Errorf("request shape wrong: %s %s accept=%q", gotMethod, gotPath, gotAccept)
	}
	if out.Value != "hello" {
		t.Errorf("Value = %q, want hello", out.Value)
	}
}

func TestDoJSON_PostEncodesBody(t *testing.T) {
	var gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	body := map[string]int{"n": 42}
	if err := c.doJSON(context.Background(), "POST", "/x", nil, body, nil); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
	if string(gotBody) != `{"n":42}` {
		t.Errorf("body = %q, want %q", gotBody, `{"n":42}`)
	}
}

func TestDoJSON_EncodesQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	q := url.Values{}
	q.Set("interval", "1")
	q.Set("unit", "Minute")
	if err := c.doJSON(context.Background(), "GET", "/y", q, nil, nil); err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	if gotQuery != "interval=1&unit=Minute" {
		t.Errorf("query = %q, want interval=1&unit=Minute", gotQuery)
	}
}

func TestDoJSON_ErrorResponse_ParsesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"Error":"BadRequest","Message":"invalid symbol"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	err := c.doJSON(context.Background(), "GET", "/z", nil, nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 || apiErr.Message != "invalid symbol" {
		t.Errorf("got %+v", apiErr)
	}
}

func TestDoJSON_ErrorResponse_FallsBackToRawBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("oops"))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	err := c.doJSON(context.Background(), "GET", "/", nil, nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 || string(apiErr.RawBody) != "oops" {
		t.Errorf("got %+v", apiErr)
	}
}

func TestRefreshAccessToken_HappyPath(t *testing.T) {
	var gotForm url.Values
	var gotCT string
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"new-access","expires_in":1200}`))
	}))
	defer tokenSrv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.tokenURL = tokenSrv.URL

	if err := c.refreshAccessToken(context.Background()); err != nil {
		t.Fatalf("refreshAccessToken: %v", err)
	}
	if c.currentAccessToken() != "new-access" {
		t.Errorf("accessToken = %q, want new-access", c.currentAccessToken())
	}
	if gotCT != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", gotCT)
	}
	if gotForm.Get("grant_type") != "refresh_token" ||
		gotForm.Get("client_id") != "id" ||
		gotForm.Get("client_secret") != "secret" ||
		gotForm.Get("refresh_token") != "refresh" {
		t.Errorf("form wrong: %v", gotForm)
	}
}

func TestRefreshAccessToken_RotatesRefreshToken(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"new-access","refresh_token":"rotated","expires_in":1200}`))
	}))
	defer tokenSrv.Close()

	var rotated string
	c := NewClient(Test, "id", "secret", "refresh",
		WithRefreshTokenRotate(func(s string) { rotated = s }))
	c.tokenURL = tokenSrv.URL

	if err := c.refreshAccessToken(context.Background()); err != nil {
		t.Fatalf("refreshAccessToken: %v", err)
	}
	if rotated != "rotated" {
		t.Errorf("onRotate received %q, want rotated", rotated)
	}
	c.mu.Lock()
	rt := c.refreshToken
	c.mu.Unlock()
	if rt != "rotated" {
		t.Errorf("in-memory refresh token = %q, want rotated", rt)
	}
}

func TestRefreshAccessToken_DoesNotCallRotateWhenSameToken(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"new","refresh_token":"refresh","expires_in":1200}`))
	}))
	defer tokenSrv.Close()

	called := false
	c := NewClient(Test, "id", "secret", "refresh",
		WithRefreshTokenRotate(func(string) { called = true }))
	c.tokenURL = tokenSrv.URL

	c.refreshAccessToken(context.Background())
	if called {
		t.Error("onRotate should not fire when refresh token unchanged")
	}
}

func TestRefreshAccessToken_ErrorResponse(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"Error":"invalid_grant","Message":"refresh token expired"}`))
	}))
	defer tokenSrv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.tokenURL = tokenSrv.URL

	err := c.refreshAccessToken(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
}
