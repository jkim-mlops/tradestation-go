package tradestation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAccountError_JSONRoundtrip(t *testing.T) {
	raw := `{"AccountID":"123","Error":"NotFound","Message":"account not found"}`
	var e AccountError
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.AccountID != "123" || e.ErrorCode != "NotFound" || e.Message != "account not found" {
		t.Errorf("decoded wrong: %+v", e)
	}
}

func TestValidateAccountIDs(t *testing.T) {
	cases := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{"nil", nil, true},
		{"empty slice", []string{}, true},
		{"single", []string{"123"}, false},
		{"25 ok", make25IDs(), false},
		{"26 too many", append(make25IDs(), "extra"), true},
		{"empty string element", []string{"123", ""}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAccountIDs(tc.ids)
			if tc.wantErr && err == nil {
				t.Errorf("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("want nil, got %v", err)
			}
		})
	}
}

func TestValidateOrderIDs(t *testing.T) {
	cases := []struct {
		name    string
		ids     []string
		wantErr bool
	}{
		{"nil", nil, true},
		{"empty slice", []string{}, true},
		{"single", []string{"order-1"}, false},
		{"many ok", []string{"a", "b", "c"}, false},
		{"empty string element", []string{"a", ""}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOrderIDs(tc.ids)
			if tc.wantErr && err == nil {
				t.Errorf("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("want nil, got %v", err)
			}
		})
	}
}

func make25IDs() []string {
	out := make([]string, 25)
	for i := range out {
		out[i] = "acct"
	}
	return out
}

func TestGetAccounts(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Accounts":[
            {"AccountID":"123","AccountType":"Cash","Currency":"USD","Status":"Active"},
            {"AccountID":"456","AccountType":"Margin","Currency":"USD","Status":"Active"}
        ]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	accounts, err := svc.GetAccounts(context.Background())
	if err != nil {
		t.Fatalf("GetAccounts: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts" {
		t.Errorf("path = %q, want /v3/brokerage/accounts", gotPath)
	}
	if gotAuth == "" {
		t.Errorf("Authorization header missing")
	}
	if len(accounts) != 2 || accounts[0].AccountID != "123" || accounts[1].AccountType != "Margin" {
		t.Errorf("decoded wrong: %+v", accounts)
	}
}

func TestGetBalances(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{
            "Balances":[{"AccountID":"123","CashBalance":"1000.50","BuyingPower":"2000"}],
            "Errors":[{"AccountID":"456","Error":"NotFound","Message":"account not found"}]
        }`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	resp, err := svc.GetBalances(context.Background(), []string{"123", "456"})
	if err != nil {
		t.Fatalf("GetBalances: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123,456/balances" {
		t.Errorf("path = %q", gotPath)
	}
	if len(resp.Balances) != 1 || resp.Balances[0].CashBalance != 1000.50 {
		t.Errorf("balances wrong: %+v", resp.Balances)
	}
	if len(resp.Errors) != 1 || resp.Errors[0].ErrorCode != "NotFound" {
		t.Errorf("errors wrong: %+v", resp.Errors)
	}
}

func TestGetBalances_ValidationRejects(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &BrokerageService{client: c}
	if _, err := svc.GetBalances(context.Background(), nil); err == nil {
		t.Error("want error for empty accountIDs")
	}
}

func TestGetBalancesBOD(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"BODBalances":[{"AccountID":"123","AccountType":"Cash"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	resp, err := svc.GetBalancesBOD(context.Background(), []string{"123"})
	if err != nil {
		t.Fatalf("GetBalancesBOD: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123/bodbalances" {
		t.Errorf("path = %q", gotPath)
	}
	if len(resp.BODBalances) != 1 || resp.BODBalances[0].AccountID != "123" {
		t.Errorf("decoded wrong: %+v", resp.BODBalances)
	}
}

func TestGetPositions_NoOpts(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"Positions":[{"AccountID":"123","Symbol":"AAPL","Quantity":"10"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	resp, err := svc.GetPositions(context.Background(), []string{"123"})
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123/positions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty", gotQuery)
	}
	if len(resp.Positions) != 1 || resp.Positions[0].Symbol != "AAPL" {
		t.Errorf("decoded wrong: %+v", resp.Positions)
	}
}

func TestGetPositions_WithSymbol(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"Positions":[]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	_, err := svc.GetPositions(context.Background(), []string{"123"}, WithSymbol("AAPL*"))
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if gotQuery != "symbol=AAPL%2A" {
		t.Errorf("query = %q, want symbol=AAPL%%2A", gotQuery)
	}
}

func TestGetOrders(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"Orders":[{"OrderID":"o1","AccountID":"123","Status":"Open"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	resp, err := svc.GetOrders(context.Background(), []string{"123"})
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123/orders" {
		t.Errorf("path = %q", gotPath)
	}
	if len(resp.Orders) != 1 || resp.Orders[0].OrderID != "o1" {
		t.Errorf("decoded wrong: %+v", resp.Orders)
	}
}

func TestGetOrdersByID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"Orders":[{"OrderID":"o1"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	_, err := svc.GetOrdersByID(context.Background(), []string{"123"}, []string{"o1", "o2"})
	if err != nil {
		t.Fatalf("GetOrdersByID: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123/orders/o1,o2" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestGetOrdersByID_ValidationRejectsEmptyOrderIDs(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &BrokerageService{client: c}
	if _, err := svc.GetOrdersByID(context.Background(), []string{"123"}, nil); err == nil {
		t.Error("want error for empty orderIDs")
	}
}

func TestGetHistoricalOrders_SinglePage(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"Orders":[{"OrderID":"o1"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	resp, err := svc.GetHistoricalOrders(context.Background(), []string{"123"}, since)
	if err != nil {
		t.Fatalf("GetHistoricalOrders: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123/historicalorders" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "since=2026-03-01" {
		t.Errorf("query = %q", gotQuery)
	}
	if len(resp.Orders) != 1 {
		t.Errorf("got %d orders, want 1", len(resp.Orders))
	}
}

func TestGetHistoricalOrders_MultiPageAggregates(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			w.Write([]byte(`{"Orders":[{"OrderID":"o1"},{"OrderID":"o2"}],"NextToken":"tok1"}`))
		case 2:
			if got := r.URL.Query().Get("nextToken"); got != "tok1" {
				t.Errorf("nextToken = %q, want tok1", got)
			}
			w.Write([]byte(`{"Orders":[{"OrderID":"o3"}],"NextToken":"tok2"}`))
		case 3:
			w.Write([]byte(`{"Orders":[{"OrderID":"o4"}]}`))
		}
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	resp, err := svc.GetHistoricalOrders(context.Background(), []string{"123"}, since)
	if err != nil {
		t.Fatalf("GetHistoricalOrders: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if len(resp.Orders) != 4 {
		t.Errorf("orders = %d, want 4", len(resp.Orders))
	}
}

func TestGetHistoricalOrders_WithMaxPages(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"Orders":[{"OrderID":"x"}],"NextToken":"keep-going"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	resp, err := svc.GetHistoricalOrders(context.Background(), []string{"123"}, since, WithMaxPages(2))
	if err != nil {
		t.Fatalf("GetHistoricalOrders: %v", err)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (capped)", calls)
	}
	if len(resp.Orders) != 2 {
		t.Errorf("orders = %d, want 2", len(resp.Orders))
	}
}

func TestGetHistoricalOrders_WithPageSize(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"Orders":[]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.GetHistoricalOrders(context.Background(), []string{"123"}, since, WithPageSize(100))
	if err != nil {
		t.Fatalf("GetHistoricalOrders: %v", err)
	}
	if gotQuery != "pageSize=100&since=2026-03-01" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestGetHistoricalOrders_ContextCancel(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"Orders":[{"OrderID":"x"}],"NextToken":"more"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.GetHistoricalOrders(ctx, []string{"123"}, since)
	if err == nil {
		t.Error("want context-cancelled error")
	}
}

func TestGetHistoricalOrdersByID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"Orders":[{"OrderID":"o1"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.GetHistoricalOrdersByID(context.Background(), []string{"123"}, []string{"o1", "o2"}, since)
	if err != nil {
		t.Fatalf("GetHistoricalOrdersByID: %v", err)
	}
	if gotPath != "/v3/brokerage/accounts/123/historicalorders/o1,o2" {
		t.Errorf("path = %q", gotPath)
	}
}
