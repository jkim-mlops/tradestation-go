//go:build integration

package tradestation

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func integrationClient(t *testing.T) *Client {
	t.Helper()
	loadEnvFile(".env")
	id := os.Getenv("TRADESTATION_CLIENT_ID")
	secret := os.Getenv("TRADESTATION_CLIENT_SECRET")
	rt := os.Getenv("TRADESTATION_REFRESH_TOKEN")
	if id == "" || secret == "" || rt == "" {
		t.Skip("TRADESTATION_CLIENT_ID / _SECRET / _REFRESH_TOKEN not set; skipping integration tests")
	}
	return NewClient(Test, id, secret, rt)
}

func TestIntegration_GetBars_Daily(t *testing.T) {
	c := integrationClient(t)
	svc := &MarketDataService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bars, err := svc.GetBars(ctx, "AAPL", GetBarsParams{
		Interval: 1, Unit: BarUnitDaily, BarsBack: 5,
	})
	if err != nil {
		t.Fatalf("GetBars: %v", err)
	}
	if len(bars) == 0 {
		t.Fatal("no bars returned")
	}
	for i, b := range bars {
		if b.High < b.Low {
			t.Errorf("bar %d: High %v < Low %v", i, b.High, b.Low)
		}
		if b.TotalVolume < 0 {
			t.Errorf("bar %d: negative volume %d", i, b.TotalVolume)
		}
	}
	for i := 1; i < len(bars); i++ {
		if !bars[i].TimeStamp.After(bars[i-1].TimeStamp) {
			t.Errorf("bars not strictly increasing: %d=%v, %d=%v", i-1, bars[i-1].TimeStamp, i, bars[i].TimeStamp)
		}
	}
}

func TestIntegration_GetBars_Minute(t *testing.T) {
	c := integrationClient(t)
	svc := &MarketDataService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bars, err := svc.GetBars(ctx, "SPY", GetBarsParams{
		Interval: 5, Unit: BarUnitMinute, BarsBack: 20,
	})
	if err != nil {
		t.Fatalf("GetBars: %v", err)
	}
	if len(bars) == 0 {
		t.Fatal("no bars returned")
	}
}

func TestIntegration_GetQuote_Single(t *testing.T) {
	c := integrationClient(t)
	svc := &MarketDataService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	quotes, err := svc.GetQuote(ctx, []string{"AAPL"})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if len(quotes) != 1 || quotes[0].Symbol != "AAPL" {
		t.Errorf("quotes = %+v", quotes)
	}
}

func TestIntegration_GetQuote_Multiple(t *testing.T) {
	c := integrationClient(t)
	svc := &MarketDataService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	syms := []string{"AAPL", "MSFT", "GOOGL", "SPY", "QQQ"}
	quotes, err := svc.GetQuote(ctx, syms)
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if len(quotes) != len(syms) {
		t.Errorf("got %d quotes, want %d", len(quotes), len(syms))
	}
	got := make(map[string]bool, len(quotes))
	for _, q := range quotes {
		got[q.Symbol] = true
	}
	for _, s := range syms {
		if !got[s] {
			t.Errorf("missing quote for %s", s)
		}
	}
}

func TestIntegration_RefreshPath(t *testing.T) {
	c := integrationClient(t)
	// Force a refresh on the next call by clearing the in-memory access token.
	c.mu.Lock()
	c.accessToken = ""
	c.mu.Unlock()

	svc := &MarketDataService{client: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := svc.GetQuote(ctx, []string{"AAPL"}); err != nil {
		t.Fatalf("GetQuote after forced refresh: %v", err)
	}
	if c.currentAccessToken() == "" {
		t.Error("access token not populated after refresh")
	}
}

func TestIntegration_BogusSymbol(t *testing.T) {
	c := integrationClient(t)
	svc := &MarketDataService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := svc.GetBars(ctx, "NOSUCHSYMBOL_XXX", GetBarsParams{
		Interval: 1, Unit: BarUnitDaily, BarsBack: 1,
	})
	if err == nil {
		t.Fatal("expected error for bogus symbol")
	}
	// We accept any error shape here - the actual TradeStation behavior may be
	// 4xx, may be 200 with empty bars. Adjust once integration run reveals truth.
}

func dumpJSON(t *testing.T, label string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Logf("%s: marshal error: %v", label, err)
		return
	}
	t.Logf("%s:\n%s", label, b)
}

func TestIntegration_GetAccounts(t *testing.T) {
	c := integrationClient(t)
	svc := &BrokerageService{client: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	accounts, err := svc.GetAccounts(ctx)
	if err != nil {
		t.Fatalf("GetAccounts: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatal("no accounts returned — sandbox account required")
	}
	dumpJSON(t, "accounts", accounts)
	for _, a := range accounts {
		if a.AccountID == "" {
			t.Errorf("account missing ID: %+v", a)
		}
	}
}

func fetchSandboxAccountIDs(t *testing.T, c *Client) []string {
	t.Helper()
	svc := &BrokerageService{client: c}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	accounts, err := svc.GetAccounts(ctx)
	if err != nil {
		t.Fatalf("fetch accounts: %v", err)
	}
	ids := make([]string, 0, len(accounts))
	for _, a := range accounts {
		ids = append(ids, a.AccountID)
	}
	if len(ids) == 0 {
		t.Skip("no accounts on sandbox — skipping dependent tests")
	}
	return ids
}

func TestIntegration_GetBalances(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)
	svc := &BrokerageService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.GetBalances(ctx, ids)
	if err != nil {
		t.Fatalf("GetBalances: %v", err)
	}
	dumpJSON(t, "balances", resp)
	if len(resp.Balances)+len(resp.Errors) != len(ids) {
		t.Errorf("balances+errors = %d, want %d", len(resp.Balances)+len(resp.Errors), len(ids))
	}
}

func TestIntegration_GetBalancesBOD(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)
	svc := &BrokerageService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.GetBalancesBOD(ctx, ids)
	if err != nil {
		t.Fatalf("GetBalancesBOD: %v", err)
	}
	dumpJSON(t, "bodBalances", resp)
}

func TestIntegration_GetPositions(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)
	svc := &BrokerageService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.GetPositions(ctx, ids)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	dumpJSON(t, "positions", resp)
}

func TestIntegration_GetOrders(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)
	svc := &BrokerageService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.GetOrders(ctx, ids)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	dumpJSON(t, "orders", resp)
}

func TestIntegration_GetHistoricalOrders(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)
	svc := &BrokerageService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	since := time.Now().AddDate(0, 0, -30)
	resp, err := svc.GetHistoricalOrders(ctx, ids, since, WithMaxPages(3))
	if err != nil {
		t.Fatalf("GetHistoricalOrders: %v", err)
	}
	dumpJSON(t, "historicalOrders", resp)
}

func TestIntegration_StreamOrders(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := c.Brokerage().StreamOrders(ctx, ids, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamOrders: %v", err)
	}

	var gotSnapshot bool
	for ev := range events {
		switch {
		case ev.Err != nil:
			t.Fatalf("stream error: %v", ev.Err)
		case ev.Data != nil:
			t.Logf("order: %+v", *ev.Data)
		case ev.Status == StreamStatusEndSnapshot:
			t.Logf("status: EndSnapshot")
			gotSnapshot = true
			cancel()
		default:
			if ev.Status != "" {
				t.Logf("status: %s", ev.Status)
			}
		}
	}
	if !gotSnapshot {
		t.Error("no EndSnapshot received")
	}
}

func TestIntegration_StreamQuotes(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	wantSyms := []string{"AAPL", "SPY"}
	events, err := c.MarketData().StreamQuotes(ctx, wantSyms, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamQuotes: %v", err)
	}

	got := make(map[string]Quote)
	for ev := range events {
		switch {
		case ev.Err != nil:
			t.Fatalf("stream error: %v", ev.Err)
		case ev.Data != nil:
			t.Logf("quote: %+v", *ev.Data)
			got[ev.Data.Symbol] = *ev.Data
		case ev.Status != "":
			t.Logf("status: %s", ev.Status)
		}
		if len(got) == len(wantSyms) {
			cancel()
			break
		}
	}

	for _, sym := range wantSyms {
		q, ok := got[sym]
		if !ok {
			t.Errorf("no quote received for %s", sym)
			continue
		}
		if q.Ask <= 0 || q.Bid <= 0 {
			t.Errorf("%s quote missing prices: %+v", sym, q)
		}
	}
}
