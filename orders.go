package tradestation

import (
	"errors"
	"fmt"
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
