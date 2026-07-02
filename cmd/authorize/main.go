package main

import (
	"bufio"
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
)

const (
	authorizeURL    = "https://signin.tradestation.com/authorize"
	tokenURL        = "https://signin.tradestation.com/oauth/token"
	audience        = "https://api.tradestation.com"
	defaultRedirect = "http://localhost:8080"
	defaultScopes   = "openid offline_access MarketData ReadAccount Trade profile"
)

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
