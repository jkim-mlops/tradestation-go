package tradestation

import (
	"net/http"
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

	http    *http.Client // wrapped in authTransport
	rawHTTP *http.Client // plain — used only for /oauth/token
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
	return t.base.RoundTrip(req)
}
