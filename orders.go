package tradestation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// OrderService bundles TradeStation Order Execution REST endpoints.
type OrderService struct {
	client *Client
}

// OrderExecution returns an OrderService bound to this client.
func (c *Client) OrderExecution() *OrderService {
	return &OrderService{client: c}
}

func validateOrderRequest(req *OrderRequest) error {
	if req.AccountID == "" {
		return errors.New("tradestation: OrderRequest.AccountID is required")
	}
	if req.Symbol == "" {
		return errors.New("tradestation: OrderRequest.Symbol is required")
	}
	if req.Quantity <= 0 {
		return errors.New("tradestation: OrderRequest.Quantity must be positive")
	}
	if req.OrderType == "" {
		return errors.New("tradestation: OrderRequest.OrderType is required")
	}
	if req.TradeAction == "" {
		return errors.New("tradestation: OrderRequest.TradeAction is required")
	}
	if req.TimeInForce.Duration == "" {
		return errors.New("tradestation: OrderRequest.TimeInForce.Duration is required")
	}
	if req.TimeInForce.Duration == DurationGTD && req.TimeInForce.ExpirationDate == "" {
		return errors.New("tradestation: Duration=GTD requires TimeInForce.ExpirationDate")
	}

	switch req.OrderType {
	case OrderTypeMarket:
		if req.LimitPrice != 0 || req.StopPrice != 0 {
			return errors.New("tradestation: Market orders must not set LimitPrice or StopPrice")
		}
	case OrderTypeLimit:
		if req.LimitPrice <= 0 {
			return errors.New("tradestation: Limit orders require positive LimitPrice")
		}
		if req.StopPrice != 0 {
			return errors.New("tradestation: Limit orders must not set StopPrice")
		}
	case OrderTypeStopMarket:
		if req.StopPrice <= 0 {
			return errors.New("tradestation: StopMarket orders require positive StopPrice")
		}
		if req.LimitPrice != 0 {
			return errors.New("tradestation: StopMarket orders must not set LimitPrice")
		}
	case OrderTypeStopLimit:
		if req.LimitPrice <= 0 || req.StopPrice <= 0 {
			return errors.New("tradestation: StopLimit orders require positive LimitPrice and StopPrice")
		}
	}

	if len(req.Legs) > 0 {
		if err := validateOrderLegs(req.Legs); err != nil {
			return err
		}
	}
	return nil
}

func validateOrderLegs(legs []OrderLegRequest) error {
	for i, leg := range legs {
		if leg.Symbol == "" {
			return fmt.Errorf("tradestation: Legs[%d].Symbol is required", i)
		}
		if leg.Quantity <= 0 {
			return fmt.Errorf("tradestation: Legs[%d].Quantity must be positive", i)
		}
		if leg.TradeAction == "" {
			return fmt.Errorf("tradestation: Legs[%d].TradeAction is required", i)
		}
	}
	return nil
}

func validateOrderGroupRequest(req *OrderGroupRequest) error {
	if req.Type == "" {
		return errors.New("tradestation: OrderGroupRequest.Type is required")
	}
	if len(req.Orders) < 2 {
		return errors.New("tradestation: OrderGroupRequest.Orders must contain at least 2 orders")
	}
	for i := range req.Orders {
		if err := validateOrderRequest(&req.Orders[i]); err != nil {
			return fmt.Errorf("tradestation: OrderGroupRequest.Orders[%d]: %w", i, err)
		}
	}
	return nil
}

func validateReplaceOrderRequest(req *ReplaceOrderRequest) error {
	if req.Quantity == 0 && req.LimitPrice == 0 && req.StopPrice == 0 &&
		req.TimeInForce == nil && req.AdvancedOptions == "" {
		return errors.New("tradestation: ReplaceOrderRequest has no modifications")
	}
	if req.Quantity < 0 || req.LimitPrice < 0 || req.StopPrice < 0 {
		return errors.New("tradestation: ReplaceOrderRequest values must be non-negative")
	}
	if req.TimeInForce != nil && req.TimeInForce.Duration == DurationGTD && req.TimeInForce.ExpirationDate == "" {
		return errors.New("tradestation: Duration=GTD requires TimeInForce.ExpirationDate")
	}
	return nil
}

func (s *OrderService) GetActivationTriggers(ctx context.Context) ([]ActivationTrigger, error) {
	var resp struct {
		ActivationTriggers []ActivationTrigger `json:"ActivationTriggers"`
	}
	if err := s.client.doJSON(ctx, "GET", "/v3/orderexecution/activationtriggers", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.ActivationTriggers, nil
}

func (s *OrderService) GetRoutes(ctx context.Context) ([]OrderRoute, error) {
	var resp struct {
		Routes []OrderRoute `json:"Routes"`
	}
	if err := s.client.doJSON(ctx, "GET", "/v3/orderexecution/routes", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Routes, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return errors.New("tradestation: CancelOrder requires an order ID")
	}
	path := "/v3/orderexecution/orders/" + url.PathEscape(orderID)
	return s.client.doJSON(ctx, "DELETE", path, nil, nil, nil)
}

func (s *OrderService) PlaceOrder(ctx context.Context, req OrderRequest) (*PlaceOrderResponse, error) {
	if err := validateOrderRequest(&req); err != nil {
		return nil, err
	}
	var out PlaceOrderResponse
	if err := s.client.doJSON(ctx, "POST", "/v3/orderexecution/orders", nil, &req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *OrderService) PlaceOrderConfirm(ctx context.Context, req OrderRequest) (*ConfirmationResponse, error) {
	if err := validateOrderRequest(&req); err != nil {
		return nil, err
	}
	var out ConfirmationResponse
	if err := s.client.doJSON(ctx, "POST", "/v3/orderexecution/orderconfirm", nil, &req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *OrderService) ReplaceOrder(ctx context.Context, orderID string, req ReplaceOrderRequest) (*Order, error) {
	if orderID == "" {
		return nil, errors.New("tradestation: ReplaceOrder requires an order ID")
	}
	if err := validateReplaceOrderRequest(&req); err != nil {
		return nil, err
	}
	path := "/v3/orderexecution/orders/" + url.PathEscape(orderID)
	var out Order
	if err := s.client.doJSON(ctx, "PUT", path, nil, &req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *OrderService) PlaceOrderGroup(ctx context.Context, req OrderGroupRequest) (*PlaceOrderResponse, error) {
	if err := validateOrderGroupRequest(&req); err != nil {
		return nil, err
	}
	var out PlaceOrderResponse
	if err := s.client.doJSON(ctx, "POST", "/v3/orderexecution/ordergroups", nil, &req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *OrderService) PlaceOrderGroupConfirm(ctx context.Context, req OrderGroupRequest) (*ConfirmationResponse, error) {
	if err := validateOrderGroupRequest(&req); err != nil {
		return nil, err
	}
	var out ConfirmationResponse
	if err := s.client.doJSON(ctx, "POST", "/v3/orderexecution/ordergroupsconfirm", nil, &req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
