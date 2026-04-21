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

func TestIntegration_StreamPositions(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := c.Brokerage().StreamPositions(ctx, ids, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamPositions: %v", err)
	}

	var gotSnapshot bool
	for ev := range events {
		switch {
		case ev.Err != nil:
			t.Fatalf("stream error: %v", ev.Err)
		case ev.Data != nil:
			t.Logf("position: %+v", *ev.Data)
		case ev.Status == StreamStatusEndSnapshot:
			t.Logf("status: EndSnapshot")
			gotSnapshot = true
			cancel()
		}
	}
	if !gotSnapshot {
		t.Error("no EndSnapshot received")
	}
}

func TestIntegration_StreamBars(t *testing.T) {
	c := integrationClient(t)

	const wantBars = 5

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := c.MarketData().StreamBars(
		ctx,
		"SPY",
		StreamBarsParams{Interval: 1, Unit: BarUnitMinute, BarsBack: wantBars},
		WithoutReconnect(),
	)
	if err != nil {
		t.Fatalf("StreamBars: %v", err)
	}

	// Bar streams don't emit EndSnapshot (unlike orders/positions/quotes).
	// We exit after receiving the requested BarsBack historical bars.
	var bars int
	for ev := range events {
		switch {
		case ev.Err != nil:
			t.Fatalf("stream error: %v", ev.Err)
		case ev.Data != nil:
			t.Logf("bar: %+v", *ev.Data)
			bars++
		case ev.Status != "":
			t.Logf("status: %s", ev.Status)
		}
		if bars >= wantBars {
			cancel()
			break
		}
	}
	if bars < wantBars {
		t.Errorf("received %d bars, want >= %d", bars, wantBars)
	}
}

func TestIntegration_StreamOrdersByID(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	getCtx, getCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer getCancel()
	resp, err := c.Brokerage().GetOrders(getCtx, ids)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	if len(resp.Orders) == 0 {
		t.Skip("no orders on sandbox — cannot test StreamOrdersByID")
	}
	orderID := resp.Orders[0].OrderID
	accountID := resp.Orders[0].AccountID
	t.Logf("streaming order %s on account %s", orderID, accountID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events, err := c.Brokerage().StreamOrdersByID(
		ctx,
		[]string{accountID},
		[]string{orderID},
		WithoutReconnect(),
	)
	if err != nil {
		t.Fatalf("StreamOrdersByID: %v", err)
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

func TestIntegration_GetActivationTriggers(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	triggers, err := c.OrderExecution().GetActivationTriggers(ctx)
	if err != nil {
		t.Fatalf("GetActivationTriggers: %v", err)
	}
	for _, at := range triggers {
		t.Logf("trigger: %+v", at)
	}
	if len(triggers) == 0 {
		t.Error("no triggers returned")
	}
}

func TestIntegration_GetRoutes(t *testing.T) {
	c := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	routes, err := c.OrderExecution().GetRoutes(ctx)
	if err != nil {
		t.Fatalf("GetRoutes: %v", err)
	}
	for _, r := range routes {
		t.Logf("route: %+v", r)
	}
	if len(routes) == 0 {
		t.Error("no routes returned")
	}
}

func TestIntegration_PlaceOrderConfirm(t *testing.T) {
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := OrderRequest{
		AccountID:   ids[0],
		Symbol:      "AAPL",
		Quantity:    1,
		OrderType:   OrderTypeLimit,
		TradeAction: TradeActionBuy,
		LimitPrice:  1, // far-below-market; preview only
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	resp, err := c.OrderExecution().PlaceOrderConfirm(ctx, req)
	if err != nil {
		t.Fatalf("PlaceOrderConfirm: %v", err)
	}
	if len(resp.Confirmations) == 0 && len(resp.Errors) == 0 {
		t.Error("empty confirmation response")
	}
	for _, c := range resp.Confirmations {
		t.Logf("confirmation: %+v", c)
	}
	for _, e := range resp.Errors {
		t.Logf("error: %+v", e)
	}
}

// requireOrderPlacementOptIn skips unless the destructive-tests env var is set.
func requireOrderPlacementOptIn(t *testing.T) {
	t.Helper()
	if os.Getenv("TRADESTATION_INTEGRATION_PLACE_ORDERS") != "1" {
		t.Skip("set TRADESTATION_INTEGRATION_PLACE_ORDERS=1 to run destructive order-placement tests")
	}
}

func TestIntegration_PlaceAndCancelOrder(t *testing.T) {
	requireOrderPlacementOptIn(t)
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := OrderRequest{
		AccountID:   ids[0],
		Symbol:      "AAPL",
		Quantity:    1,
		OrderType:   OrderTypeLimit,
		TradeAction: TradeActionBuy,
		LimitPrice:  1, // far-below-market, won't fill
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	resp, err := c.OrderExecution().PlaceOrder(ctx, req)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	t.Logf("placed: %+v, errors: %+v", resp.Orders, resp.Errors)
	if len(resp.Orders) == 0 {
		t.Fatalf("no order placed: errors=%+v", resp.Errors)
	}
	orderID := resp.Orders[0].OrderID

	// Cleanup: cancel in defer so panics don't leak orders.
	defer func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		if err := c.OrderExecution().CancelOrder(cctx, orderID); err != nil {
			t.Logf("cancel cleanup failed for %s: %v", orderID, err)
		}
	}()

	// Verify order is visible via GetOrders.
	orders, err := c.Brokerage().GetOrders(ctx, ids)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	var found bool
	for _, o := range orders.Orders {
		if o.OrderID == orderID {
			t.Logf("found order: %+v", o)
			found = true
			break
		}
	}
	if !found {
		t.Errorf("placed order %s not found in GetOrders", orderID)
	}
}

func TestIntegration_PlaceAndReplaceAndCancel(t *testing.T) {
	requireOrderPlacementOptIn(t)
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := OrderRequest{
		AccountID:   ids[0],
		Symbol:      "AAPL",
		Quantity:    1,
		OrderType:   OrderTypeLimit,
		TradeAction: TradeActionBuy,
		LimitPrice:  1,
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	resp, err := c.OrderExecution().PlaceOrder(ctx, req)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if len(resp.Orders) == 0 {
		t.Fatalf("no order placed: errors=%+v", resp.Errors)
	}
	orderID := resp.Orders[0].OrderID

	defer func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		if err := c.OrderExecution().CancelOrder(cctx, orderID); err != nil {
			t.Logf("cancel cleanup failed for %s: %v", orderID, err)
		}
	}()

	replaced, err := c.OrderExecution().ReplaceOrder(ctx, orderID, ReplaceOrderRequest{Quantity: 2})
	if err != nil {
		t.Fatalf("ReplaceOrder: %v", err)
	}
	t.Logf("replaced: %+v", replaced)
}

func TestIntegration_PlaceAndCancelOCOGroup(t *testing.T) {
	requireOrderPlacementOptIn(t)
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	legA := OrderRequest{
		AccountID: ids[0], Symbol: "AAPL", Quantity: 1,
		OrderType: OrderTypeLimit, TradeAction: TradeActionBuy,
		LimitPrice:  1, // won't fill
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	legB := legA
	legB.LimitPrice = 2 // also won't fill; different price so distinguishable

	group := OrderGroupRequest{Type: OrderGroupTypeOCO, Orders: []OrderRequest{legA, legB}}
	resp, err := c.OrderExecution().PlaceOrderGroup(ctx, group)
	if err != nil {
		t.Fatalf("PlaceOrderGroup: %v", err)
	}
	t.Logf("placed: %+v, errors: %+v", resp.Orders, resp.Errors)
	if len(resp.Orders) < 2 {
		t.Fatalf("expected 2 orders, got %d (errors: %+v)", len(resp.Orders), resp.Errors)
	}

	defer func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		for _, o := range resp.Orders {
			if err := c.OrderExecution().CancelOrder(cctx, o.OrderID); err != nil {
				t.Logf("cancel cleanup failed for %s: %v", o.OrderID, err)
			}
		}
	}()

	// Verify both visible in GetOrders.
	orders, err := c.Brokerage().GetOrders(ctx, ids)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	seen := make(map[string]bool)
	for _, o := range orders.Orders {
		for _, placed := range resp.Orders {
			if o.OrderID == placed.OrderID {
				seen[o.OrderID] = true
				t.Logf("found order: %+v", o)
			}
		}
	}
	if len(seen) != len(resp.Orders) {
		t.Errorf("saw %d of %d placed orders in GetOrders", len(seen), len(resp.Orders))
	}
}

func TestIntegration_PlaceAndCancelBracketGroup(t *testing.T) {
	requireOrderPlacementOptIn(t)
	c := integrationClient(t)
	ids := fetchSandboxAccountIDs(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	parent := OrderRequest{
		AccountID: ids[0], Symbol: "AAPL", Quantity: 1,
		OrderType: OrderTypeLimit, TradeAction: TradeActionBuy,
		LimitPrice:  1, // won't fill
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	profit := OrderRequest{
		AccountID: ids[0], Symbol: "AAPL", Quantity: 1,
		OrderType: OrderTypeLimit, TradeAction: TradeActionSell,
		LimitPrice:  500, // unreachable profit target
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	stop := OrderRequest{
		AccountID: ids[0], Symbol: "AAPL", Quantity: 1,
		OrderType: OrderTypeStopMarket, TradeAction: TradeActionSell,
		StopPrice:   0.5, // unreachable stop
		TimeInForce: TimeInForce{Duration: DurationDay},
	}

	group := OrderGroupRequest{
		Type:   OrderGroupTypeBracket,
		Orders: []OrderRequest{parent, profit, stop},
	}
	resp, err := c.OrderExecution().PlaceOrderGroup(ctx, group)
	if err != nil {
		t.Fatalf("PlaceOrderGroup: %v", err)
	}
	t.Logf("placed: %+v, errors: %+v", resp.Orders, resp.Errors)
	if len(resp.Orders) < 3 {
		t.Fatalf("expected 3 orders, got %d (errors: %+v)", len(resp.Orders), resp.Errors)
	}

	defer func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		for _, o := range resp.Orders {
			if err := c.OrderExecution().CancelOrder(cctx, o.OrderID); err != nil {
				t.Logf("cancel cleanup failed for %s: %v", o.OrderID, err)
			}
		}
	}()

	// Verify all three visible in GetOrders.
	orders, err := c.Brokerage().GetOrders(ctx, ids)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	seen := make(map[string]bool)
	for _, o := range orders.Orders {
		for _, placed := range resp.Orders {
			if o.OrderID == placed.OrderID {
				seen[o.OrderID] = true
				t.Logf("found order: %+v", o)
			}
		}
	}
	if len(seen) != len(resp.Orders) {
		t.Errorf("saw %d of %d placed orders in GetOrders", len(seen), len(resp.Orders))
	}
}
