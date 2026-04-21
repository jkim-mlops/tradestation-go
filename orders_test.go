package tradestation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validOrder() OrderRequest {
	return OrderRequest{
		AccountID:   "123",
		Symbol:      "AAPL",
		Quantity:    10,
		OrderType:   OrderTypeLimit,
		TradeAction: TradeActionBuy,
		LimitPrice:  150,
		TimeInForce: TimeInForce{Duration: DurationDay},
	}
}

func TestValidateOrderRequest_HappyPath(t *testing.T) {
	req := validOrder()
	if err := validateOrderRequest(&req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateOrderRequest_RequiresAccountID(t *testing.T) {
	req := validOrder()
	req.AccountID = ""
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for empty AccountID")
	}
}

func TestValidateOrderRequest_RequiresSymbol(t *testing.T) {
	req := validOrder()
	req.Symbol = ""
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for empty Symbol")
	}
}

func TestValidateOrderRequest_RequiresPositiveQuantity(t *testing.T) {
	req := validOrder()
	req.Quantity = 0
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for zero Quantity")
	}
	req.Quantity = -1
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for negative Quantity")
	}
}

func TestValidateOrderRequest_RequiresOrderType(t *testing.T) {
	req := validOrder()
	req.OrderType = ""
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for empty OrderType")
	}
}

func TestValidateOrderRequest_RequiresTradeAction(t *testing.T) {
	req := validOrder()
	req.TradeAction = ""
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for empty TradeAction")
	}
}

func TestValidateOrderRequest_RequiresDuration(t *testing.T) {
	req := validOrder()
	req.TimeInForce.Duration = ""
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for empty Duration")
	}
}

func TestValidateOrderRequest_GTDRequiresExpirationDate(t *testing.T) {
	req := validOrder()
	req.TimeInForce = TimeInForce{Duration: DurationGTD}
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for GTD without ExpirationDate")
	}
}

func TestValidateOrderRequest_MarketRejectsPrices(t *testing.T) {
	req := validOrder()
	req.OrderType = OrderTypeMarket
	req.LimitPrice = 150
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for Market with LimitPrice")
	}

	req = validOrder()
	req.OrderType = OrderTypeMarket
	req.LimitPrice = 0
	req.StopPrice = 150
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for Market with StopPrice")
	}
}

func TestValidateOrderRequest_LimitRequiresLimitPrice(t *testing.T) {
	req := validOrder()
	req.OrderType = OrderTypeLimit
	req.LimitPrice = 0
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for Limit without LimitPrice")
	}
}

func TestValidateOrderRequest_LimitRejectsStopPrice(t *testing.T) {
	req := validOrder()
	req.OrderType = OrderTypeLimit
	req.LimitPrice = 150
	req.StopPrice = 140
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for Limit with StopPrice")
	}
}

func TestValidateOrderRequest_StopMarketRequiresStopPrice(t *testing.T) {
	req := validOrder()
	req.OrderType = OrderTypeStopMarket
	req.LimitPrice = 0
	req.StopPrice = 0
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for StopMarket without StopPrice")
	}
}

func TestValidateOrderRequest_StopLimitRequiresBoth(t *testing.T) {
	req := validOrder()
	req.OrderType = OrderTypeStopLimit
	req.LimitPrice = 150
	req.StopPrice = 0
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for StopLimit without StopPrice")
	}
	req.LimitPrice = 0
	req.StopPrice = 140
	if err := validateOrderRequest(&req); err == nil {
		t.Error("want error for StopLimit without LimitPrice")
	}
}

func TestValidateOrderLegs_EmptySymbol(t *testing.T) {
	legs := []OrderLegRequest{{Quantity: 1, TradeAction: TradeActionBuyToOpen}}
	if err := validateOrderLegs(legs); err == nil {
		t.Error("want error for missing Symbol")
	}
}

func TestValidateOrderLegs_NonPositiveQuantity(t *testing.T) {
	legs := []OrderLegRequest{{Symbol: "X", Quantity: 0, TradeAction: TradeActionBuyToOpen}}
	if err := validateOrderLegs(legs); err == nil {
		t.Error("want error for zero Quantity")
	}
}

func TestValidateOrderLegs_MissingTradeAction(t *testing.T) {
	legs := []OrderLegRequest{{Symbol: "X", Quantity: 1}}
	if err := validateOrderLegs(legs); err == nil {
		t.Error("want error for missing TradeAction")
	}
}

func TestValidateOrderGroupRequest_RequiresType(t *testing.T) {
	g := OrderGroupRequest{Orders: []OrderRequest{validOrder(), validOrder()}}
	if err := validateOrderGroupRequest(&g); err == nil {
		t.Error("want error for empty Type")
	}
}

func TestValidateOrderGroupRequest_RequiresTwoOrders(t *testing.T) {
	g := OrderGroupRequest{Type: OrderGroupTypeOCO, Orders: []OrderRequest{validOrder()}}
	if err := validateOrderGroupRequest(&g); err == nil {
		t.Error("want error for single-order group")
	}
}

func TestValidateOrderGroupRequest_PropagatesPerOrderErrors(t *testing.T) {
	bad := validOrder()
	bad.Symbol = ""
	g := OrderGroupRequest{Type: OrderGroupTypeOCO, Orders: []OrderRequest{validOrder(), bad}}
	if err := validateOrderGroupRequest(&g); err == nil {
		t.Error("want error from per-order validation")
	}
}

func TestValidateReplaceOrderRequest_RequiresModification(t *testing.T) {
	if err := validateReplaceOrderRequest(&ReplaceOrderRequest{}); err == nil {
		t.Error("want error for empty modifications")
	}
}

func TestValidateReplaceOrderRequest_RejectsNegatives(t *testing.T) {
	if err := validateReplaceOrderRequest(&ReplaceOrderRequest{Quantity: -1}); err == nil {
		t.Error("want error for negative Quantity")
	}
	if err := validateReplaceOrderRequest(&ReplaceOrderRequest{LimitPrice: -1}); err == nil {
		t.Error("want error for negative LimitPrice")
	}
	if err := validateReplaceOrderRequest(&ReplaceOrderRequest{StopPrice: -1}); err == nil {
		t.Error("want error for negative StopPrice")
	}
}

func TestValidateReplaceOrderRequest_GTDRequiresExpirationDate(t *testing.T) {
	req := ReplaceOrderRequest{TimeInForce: &TimeInForce{Duration: DurationGTD}}
	if err := validateReplaceOrderRequest(&req); err == nil {
		t.Error("want error for GTD without ExpirationDate")
	}
}

func TestValidateReplaceOrderRequest_AcceptsValidModification(t *testing.T) {
	req := ReplaceOrderRequest{Quantity: 10}
	if err := validateReplaceOrderRequest(&req); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetActivationTriggers(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Write([]byte(`{"ActivationTriggers":[
            {"Key":"STT","Name":"Single Trade Tick","Description":"..."},
            {"Key":"DTT","Name":"Double Trade Tick","Description":"..."}
        ]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	triggers, err := c.OrderExecution().GetActivationTriggers(context.Background())
	if err != nil {
		t.Fatalf("GetActivationTriggers: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/v3/orderexecution/activationtriggers" {
		t.Errorf("path = %q", gotPath)
	}
	if len(triggers) != 2 || triggers[0].Key != "STT" {
		t.Errorf("decoded wrong: %+v", triggers)
	}
}

func TestGetRoutes(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"Routes":[
            {"Id":"Intelligent","Name":"Intelligent","AssetTypes":["STOCK","STOCKOPTION"]},
            {"Id":"NYSE","Name":"NYSE","AssetTypes":["STOCK"]}
        ]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	routes, err := c.OrderExecution().GetRoutes(context.Background())
	if err != nil {
		t.Fatalf("GetRoutes: %v", err)
	}
	if gotPath != "/v3/orderexecution/routes" {
		t.Errorf("path = %q", gotPath)
	}
	if len(routes) != 2 || routes[0].ID != "Intelligent" {
		t.Errorf("decoded wrong: %+v", routes)
	}
}

func TestCancelOrder_RequestShape(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Write([]byte(`{"OrderID":"1184080","Message":"Cancel submitted"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	if err := c.OrderExecution().CancelOrder(context.Background(), "1184080"); err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/v3/orderexecution/orders/1184080" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestCancelOrder_RejectsEmptyID(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	if err := c.OrderExecution().CancelOrder(context.Background(), ""); err == nil {
		t.Error("want error for empty orderID")
	}
}

func TestPlaceOrder_RequestShape(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"Orders":[{"OrderID":"1184080","Message":"Order placed"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	resp, err := c.OrderExecution().PlaceOrder(context.Background(), validOrder())
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if gotMethod != "POST" || gotPath != "/v3/orderexecution/orders" {
		t.Errorf("method=%q path=%q", gotMethod, gotPath)
	}

	// Verify body encoded Quantity and LimitPrice as JSON strings.
	var decoded map[string]any
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if decoded["Quantity"] != "10" {
		t.Errorf("Quantity = %v, want \"10\"", decoded["Quantity"])
	}
	if decoded["LimitPrice"] != "150" {
		t.Errorf("LimitPrice = %v, want \"150\"", decoded["LimitPrice"])
	}

	if len(resp.Orders) != 1 || resp.Orders[0].OrderID != "1184080" {
		t.Errorf("response wrong: %+v", resp)
	}
}

func TestPlaceOrder_PartialError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
            "Orders":[{"OrderID":"1","Message":"ok"}],
            "Errors":[{"OrderNumber":"0","Error":"InsufficientBuyingPower","Message":"denied"}]
        }`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	resp, err := c.OrderExecution().PlaceOrder(context.Background(), validOrder())
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if len(resp.Orders) != 1 || len(resp.Errors) != 1 {
		t.Fatalf("want 1 order + 1 error: %+v", resp)
	}
	if resp.Errors[0].ErrorCode != "InsufficientBuyingPower" {
		t.Errorf("ErrorCode = %q", resp.Errors[0].ErrorCode)
	}
}

func TestPlaceOrder_ValidationBeforeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	req := validOrder()
	req.AccountID = ""
	if _, err := c.OrderExecution().PlaceOrder(context.Background(), req); err == nil {
		t.Error("want validation error")
	}
}

func TestPlaceOrderConfirm_RequestShape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"Confirmations":[{"OrderConfirmID":"abc","Route":"Intelligent","Duration":"DAY","Account":"123","SummaryMessage":"ok","EstimatedCommission":"0","EstimatedPrice":"150","EstimatedCost":"1500","DebitCreditEstimatedCost":"-1500"}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	resp, err := c.OrderExecution().PlaceOrderConfirm(context.Background(), validOrder())
	if err != nil {
		t.Fatalf("PlaceOrderConfirm: %v", err)
	}
	if gotPath != "/v3/orderexecution/orderconfirm" {
		t.Errorf("path = %q", gotPath)
	}
	if len(resp.Confirmations) != 1 || resp.Confirmations[0].OrderConfirmID != "abc" {
		t.Errorf("decoded wrong: %+v", resp)
	}
}

func TestReplaceOrder_RequestShape(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Write([]byte(`{"OrderID":"1184080","AccountID":"123","Status":"Open"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	o, err := c.OrderExecution().ReplaceOrder(context.Background(), "1184080", ReplaceOrderRequest{Quantity: 20})
	if err != nil {
		t.Fatalf("ReplaceOrder: %v", err)
	}
	if gotMethod != "PUT" {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/v3/orderexecution/orders/1184080" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(string(gotBody), `"Quantity":"20"`) {
		t.Errorf("body did not encode Quantity as string: %s", string(gotBody))
	}
	if o.OrderID != "1184080" {
		t.Errorf("response decoded wrong: %+v", o)
	}
}

func TestReplaceOrder_RejectsEmptyID(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	if _, err := c.OrderExecution().ReplaceOrder(context.Background(), "", ReplaceOrderRequest{Quantity: 1}); err == nil {
		t.Error("want error for empty orderID")
	}
}

func TestReplaceOrder_ValidationBeforeHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	if _, err := c.OrderExecution().ReplaceOrder(context.Background(), "1", ReplaceOrderRequest{}); err == nil {
		t.Error("want validation error for empty mods")
	}
}
