package tradestation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const maxAccountIDsPerRequest = 25

type BrokerageService struct {
	client *Client
}

func validateAccountIDs(ids []string) error {
	if len(ids) == 0 {
		return errors.New("tradestation: at least one account ID required")
	}
	if len(ids) > maxAccountIDsPerRequest {
		return fmt.Errorf("tradestation: at most %d account IDs per request (got %d)", maxAccountIDsPerRequest, len(ids))
	}
	for i, id := range ids {
		if id == "" {
			return fmt.Errorf("tradestation: account ID at index %d is empty", i)
		}
	}
	return nil
}

func validateOrderIDs(ids []string) error {
	if len(ids) == 0 {
		return errors.New("tradestation: at least one order ID required")
	}
	for i, id := range ids {
		if id == "" {
			return fmt.Errorf("tradestation: order ID at index %d is empty", i)
		}
	}
	return nil
}

func (s *BrokerageService) GetAccounts(ctx context.Context) ([]Account, error) {
	var resp struct {
		Accounts []Account `json:"Accounts"`
	}
	if err := s.client.doJSON(ctx, "GET", "/v3/brokerage/accounts", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Accounts, nil
}

func (s *BrokerageService) GetBalances(ctx context.Context, accountIDs []string) (*BalancesResponse, error) {
	if err := validateAccountIDs(accountIDs); err != nil {
		return nil, err
	}
	path := "/v3/brokerage/accounts/" + strings.Join(accountIDs, ",") + "/balances"
	var out BalancesResponse
	if err := s.client.doJSON(ctx, "GET", path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *BrokerageService) GetBalancesBOD(ctx context.Context, accountIDs []string) (*BalancesBODResponse, error) {
	if err := validateAccountIDs(accountIDs); err != nil {
		return nil, err
	}
	path := "/v3/brokerage/accounts/" + strings.Join(accountIDs, ",") + "/bodbalances"
	var out BalancesBODResponse
	if err := s.client.doJSON(ctx, "GET", path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type positionsOpts struct {
	symbol string
}

type PositionsOption func(*positionsOpts)

func WithSymbol(pattern string) PositionsOption {
	return func(o *positionsOpts) { o.symbol = pattern }
}

func (s *BrokerageService) GetPositions(ctx context.Context, accountIDs []string, opts ...PositionsOption) (*PositionsResponse, error) {
	if err := validateAccountIDs(accountIDs); err != nil {
		return nil, err
	}
	cfg := positionsOpts{}
	for _, o := range opts {
		o(&cfg)
	}
	var q url.Values
	if cfg.symbol != "" {
		q = url.Values{}
		q.Set("symbol", cfg.symbol)
	}
	path := "/v3/brokerage/accounts/" + strings.Join(accountIDs, ",") + "/positions"
	var out PositionsResponse
	if err := s.client.doJSON(ctx, "GET", path, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *BrokerageService) GetOrders(ctx context.Context, accountIDs []string) (*OrdersResponse, error) {
	if err := validateAccountIDs(accountIDs); err != nil {
		return nil, err
	}
	path := "/v3/brokerage/accounts/" + strings.Join(accountIDs, ",") + "/orders"
	var out OrdersResponse
	if err := s.client.doJSON(ctx, "GET", path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
