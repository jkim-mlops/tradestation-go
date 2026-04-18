package tradestation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
