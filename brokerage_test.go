package tradestation

import (
	"encoding/json"
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
