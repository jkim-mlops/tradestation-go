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
