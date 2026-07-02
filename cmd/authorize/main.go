package main

import (
	"bufio"
	"context"
	"crypto/rand"
	_ "embed"
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
)

const (
	authorizeURL    = "https://signin.tradestation.com/authorize"
	tokenURL        = "https://signin.tradestation.com/oauth/token"
	audience        = "https://api.tradestation.com"
	defaultRedirect = "http://localhost:8080"
	defaultScopes   = "openid offline_access MarketData ReadAccount Trade profile"
)

//go:embed tradestation.svg
var logoSVG string

// completionPage is shown in the browser once the OAuth callback lands. The
// %s is replaced with the inline TradeStation logo.
const completionPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>TradeStation authorization complete</title>
<style>
  html, body { height: 100%%; margin: 0; }
  body {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: #262626;
    color: rgba(255, 255, 255, 0.85);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    text-align: center;
  }
  .logo { width: 300px; max-width: 70vw; margin-bottom: 2rem; }
  .logo svg { width: 100%%; height: auto; display: block; }
  h1 { font-weight: 500; font-size: 1.4rem; margin: 0 0 0.5rem; }
  p { font-weight: 300; font-size: 1rem; color: rgba(255, 255, 255, 0.6); margin: 0; }
</style>
</head>
<body>
  <div class="logo">%s</div>
  <h1>Authorization complete</h1>
  <p>You can close this tab.</p>
</body>
</html>`

func main() {
	idFlag := flag.String("id", "", "OAuth client ID (overrides TRADESTATION_CLIENT_ID)")
	secretFlag := flag.String("secret", "", "OAuth client secret (overrides TRADESTATION_CLIENT_SECRET)")
	scopes := flag.String("scopes", defaultScopes, "OAuth scopes (space-separated)")
	envFile := flag.String("env", ".env", "path to .env file (values take precedence over the OS environment)")
	flag.Parse()

	loadDotenv(*envFile)

	clientID := credential("client ID", *idFlag, "TRADESTATION_CLIENT_ID")
	clientSecret := credential("client secret", *secretFlag, "TRADESTATION_CLIENT_SECRET")

	ctx := context.Background()

	state := randomState()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("bind :8080: %v", err)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, completionPage, logoSVG)
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

	fmt.Println()
	fmt.Println("Refresh token (store this securely, e.g. TRADESTATION_REFRESH_TOKEN):")
	fmt.Println(tokenResp.RefreshToken)
}

// loadDotenv reads KEY=VALUE lines from path into the process environment.
// Values in the file take precedence over any already-set variables. A missing
// file is not an error — the secrets may already be exported.
func loadDotenv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		os.Setenv(key, val)
	}
}

// credential resolves a secret from the flag value, falling back to the named
// environment variable. It exits if neither source provides a value.
func credential(label, flagVal, envVar string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	log.Fatalf("no %s: pass the flag or set %s", label, envVar)
	return ""
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
