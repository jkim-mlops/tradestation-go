//go:build integration

package tradestation

import (
	"context"
	"os"
	"testing"
	"time"
)

func integrationClient(t *testing.T) *Client {
	t.Helper()
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
