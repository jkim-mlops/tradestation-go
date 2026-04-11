package tradestation

import "net/http"

type Client struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	HTTPClient   *http.Client
}

func NewClient(clientID, clientSecret, refreshToken string) *Client {
	panic("not implemented")
}

func (c *Client) Authenticate() error {
	panic("not implemented")
}

func (c *Client) refreshAccessToken() error {
	panic("not implemented")
}
