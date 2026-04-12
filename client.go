package tradestation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Environment selects the TradeStation API base URL.
type Environment int

const (
	Test       Environment = iota // sim-api.tradestation.com
	Production                    // api.tradestation.com
)

const (
	testAPIBase   = "https://sim-api.tradestation.com"
	prodAPIBase   = "https://api.tradestation.com"
	signinBaseURL = "https://signin.tradestation.com"
)

type Client struct {
	apiBase      string
	clientID     string
	clientSecret string

	mu           sync.Mutex // guards accessToken + refreshToken
	accessToken  string
	refreshToken string

	onRotate func(newToken string)

	http     *http.Client // wrapped in authTransport
	rawHTTP  *http.Client // plain — used only for /oauth/token
	tokenURL string       // override in tests; defaults to signinBaseURL+"/oauth/token"
}

type Option func(*Client)

// WithHTTPClient lets callers supply a custom *http.Client. Its Transport
// will be wrapped with the library's authTransport so refresh-on-401 still
// applies.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h.Transport == nil {
			h.Transport = http.DefaultTransport
		}
		c.http = h
	}
}

// WithRefreshTokenRotate registers a callback invoked when TradeStation
// returns a new refresh token in response to a refresh request. Useful
// when the API key is configured with rotating refresh tokens.
func WithRefreshTokenRotate(fn func(newToken string)) Option {
	return func(c *Client) { c.onRotate = fn }
}

func NewClient(env Environment, clientID, clientSecret, refreshToken string, opts ...Option) *Client {
	c := &Client{
		apiBase:      apiBaseForEnvironment(env),
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		http:         &http.Client{Transport: http.DefaultTransport},
		rawHTTP:      &http.Client{},
		tokenURL:     signinBaseURL + "/oauth/token",
	}
	for _, opt := range opts {
		opt(c)
	}
	// Wrap the final transport (default or caller-supplied) with authTransport.
	c.http.Transport = &authTransport{base: c.http.Transport, client: c}
	return c
}

func apiBaseForEnvironment(env Environment) string {
	if env == Production {
		return prodAPIBase
	}
	return testAPIBase
}

// currentAccessToken returns the current access token under the client's
// mutex. Safe to call from any goroutine.
func (c *Client) currentAccessToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.accessToken
}

// authTransport is defined here so NewClient can reference it; the full
// RoundTrip implementation lands in Task 5.
type authTransport struct {
	base   http.RoundTripper
	client *Client
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())

	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("buffer request body: %w", err)
		}
		body = b
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	req.Header.Set("Authorization", "Bearer "+t.client.currentAccessToken())
	resp, err := t.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	resp.Body.Close()
	if err := t.client.refreshAccessToken(req.Context()); err != nil {
		return nil, err
	}

	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	req.Header.Set("Authorization", "Bearer "+t.client.currentAccessToken())
	return t.base.RoundTrip(req)
}

func (c *Client) doJSON(
	ctx context.Context,
	method, path string,
	query url.Values,
	body, out any,
) error {
	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = b
	}

	u := c.apiBase + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if reqBody != nil {
		bodyReader = bytes.NewReader(reqBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func parseAPIError(resp *http.Response) error {
	raw, _ := io.ReadAll(resp.Body)
	apiErr := &APIError{StatusCode: resp.StatusCode, RawBody: raw}
	var parsed struct {
		Error   string `json:"Error"`
		Message string `json:"Message"`
	}
	if json.Unmarshal(raw, &parsed) == nil {
		switch {
		case parsed.Message != "":
			apiErr.Message = parsed.Message
		case parsed.Error != "":
			apiErr.Message = parsed.Error
		}
	}
	return apiErr
}

func (c *Client) refreshAccessToken(ctx context.Context) error {
	c.mu.Lock()
	rt := c.refreshToken
	c.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("refresh_token", rt)

	req, err := http.NewRequestWithContext(ctx, "POST", c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.rawHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAPIError(resp)
	}

	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode refresh response: %w", err)
	}

	c.mu.Lock()
	c.accessToken = out.AccessToken
	var rotated string
	if out.RefreshToken != "" && out.RefreshToken != c.refreshToken {
		c.refreshToken = out.RefreshToken
		rotated = out.RefreshToken
	}
	onRotate := c.onRotate
	c.mu.Unlock()

	if rotated != "" && onRotate != nil {
		onRotate(rotated)
	}
	return nil
}
