package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

const (
	authorizeURL      = "https://signin.tradestation.com/authorize"
	tokenURL          = "https://signin.tradestation.com/oauth/token"
	audience          = "https://api.tradestation.com"
	defaultRedirect   = "http://localhost:3000/callback"
	defaultScopes     = "openid offline_access MarketData ReadAccount Trade profile"
	defaultRefreshSSM = "/tradestation/refresh-token"
)

func main() {
	idParam := flag.String("id", "/tradestation/client-id", "SSM parameter name for client ID")
	secretParam := flag.String("secret", "/tradestation/client-secret", "SSM parameter name for client secret")
	refreshParam := flag.String("refresh", defaultRefreshSSM, "SSM parameter name to write refresh token to")
	scopes := flag.String("scopes", defaultScopes, "OAuth scopes (space-separated)")
	flag.Parse()

	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "joe-prod"
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		log.Fatalf("load aws config (profile=%s): %v", profile, err)
	}
	ssmClient := ssm.NewFromConfig(cfg)

	clientID := mustReadSSM(ctx, ssmClient, *idParam)
	clientSecret := mustReadSSM(ctx, ssmClient, *secretParam)

	state := randomState()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalf("bind :3000: %v", err)
	}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("state"); got != state {
			err := fmt.Errorf("state mismatch: got %q, want %q", got, state)
			http.Error(w, err.Error(), http.StatusBadRequest)
			errCh <- err
			return
		}
		if errStr := q.Get("error"); errStr != "" {
			desc := q.Get("error_description")
			err := fmt.Errorf("authorize error: %s %s", errStr, desc)
			http.Error(w, err.Error(), http.StatusBadRequest)
			errCh <- err
			return
		}
		code := q.Get("code")
		if code == "" {
			err := fmt.Errorf("no code in callback")
			http.Error(w, err.Error(), http.StatusBadRequest)
			errCh <- err
			return
		}
		fmt.Fprintln(w, "<html><body><h2>TradeStation authorization complete.</h2><p>You can close this tab.</p></body></html>")
		codeCh <- code
	})

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	authURL := authorizeURL + "?" + url.Values{
		"response_type": {"code"},
		"client_id":     {clientID},
		"audience":      {audience},
		"redirect_uri":  {defaultRedirect},
		"scope":         {*scopes},
		"state":         {state},
	}.Encode()

	fmt.Println("Opening browser for TradeStation authorization...")
	fmt.Println("If it doesn't open, visit this URL manually:")
	fmt.Println(authURL)
	_ = openBrowser(authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		log.Fatalf("authorize: %v", err)
	case <-time.After(5 * time.Minute):
		log.Fatalf("authorize: timed out waiting for callback")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)

	tokenResp, err := exchangeCode(ctx, clientID, clientSecret, code)
	if err != nil {
		log.Fatalf("exchange code: %v", err)
	}
	if tokenResp.RefreshToken == "" {
		log.Fatalf("no refresh_token in response — did you include the 'offline_access' scope?")
	}

	writeSSM(ctx, ssmClient, *refreshParam, tokenResp.RefreshToken)
	fmt.Printf("Wrote refresh token to SSM parameter %s\n", *refreshParam)
}

func mustReadSSM(ctx context.Context, c *ssm.Client, name string) string {
	out, err := c.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		log.Fatalf("read %s: %v", name, err)
	}
	return aws.ToString(out.Parameter.Value)
}

func writeSSM(ctx context.Context, c *ssm.Client, name, value string) {
	_, err := c.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      ssmtypes.ParameterTypeSecureString,
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		log.Fatalf("write %s: %v", name, err)
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func exchangeCode(ctx context.Context, clientID, clientSecret, code string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", defaultRedirect)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token endpoint: status %d: %s", resp.StatusCode, body)
	}

	var out tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func randomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "linux":
		return exec.Command("xdg-open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	}
	return fmt.Errorf("unsupported platform")
}
