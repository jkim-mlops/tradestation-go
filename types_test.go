package tradestation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStringFloat64_MarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   StringFloat64
		want string
	}{
		{"ten and a half", 10.5, `"10.5"`},
		{"zero", 0, `"0"`},
		{"negative", -3.25, `"-3.25"`},
		{"large", 1234567.89, `"1234567.89"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", string(got), tc.want)
			}
		})
	}
}

func TestStringFloat64_Roundtrip(t *testing.T) {
	orig := StringFloat64(42.75)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got StringFloat64
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != orig {
		t.Errorf("got %v, want %v", got, orig)
	}
}

func TestStringInt64_MarshalJSON(t *testing.T) {
	got, err := json.Marshal(StringInt64(100))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != `"100"` {
		t.Errorf("got %s, want \"100\"", string(got))
	}
}

func TestStringInt64_Roundtrip(t *testing.T) {
	orig := StringInt64(-9999)
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got StringInt64
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != orig {
		t.Errorf("got %v, want %v", got, orig)
	}
}

func TestOrderType_JSONRoundtrip(t *testing.T) {
	orig := OrderTypeLimit
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(b) != `"Limit"` {
		t.Errorf("Marshal got %s, want \"Limit\"", string(b))
	}
	var got OrderType
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != orig {
		t.Errorf("Unmarshal got %q, want %q", got, orig)
	}
}

func TestOrderRequest_MarshalEncodesNumericsAsStrings(t *testing.T) {
	req := OrderRequest{
		AccountID:   "123",
		Symbol:      "AAPL",
		Quantity:    10,
		OrderType:   OrderTypeLimit,
		TradeAction: TradeActionBuy,
		LimitPrice:  150.5,
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	// Quantity and LimitPrice should both be JSON strings.
	for _, substr := range []string{
		`"Quantity":"10"`,
		`"LimitPrice":"150.5"`,
		`"OrderType":"Limit"`,
		`"TradeAction":"BUY"`,
		`"TimeInForce":{"Duration":"DAY"}`,
	} {
		if !strings.Contains(got, substr) {
			t.Errorf("missing %q in %s", substr, got)
		}
	}
	// StopPrice was zero — should be omitted.
	if strings.Contains(got, "StopPrice") {
		t.Errorf("StopPrice should be omitted: %s", got)
	}
}

func TestTimeInForce_GTDRoundtrip(t *testing.T) {
	tif := TimeInForce{Duration: DurationGTD, ExpirationDate: "2026-12-31"}
	b, err := json.Marshal(tif)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"Duration":"GTD","ExpirationDate":"2026-12-31"}`
	if string(b) != want {
		t.Errorf("got %s, want %s", string(b), want)
	}
	var got TimeInForce
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Duration != DurationGTD || got.ExpirationDate != "2026-12-31" {
		t.Errorf("roundtrip lost data: %+v", got)
	}
}
