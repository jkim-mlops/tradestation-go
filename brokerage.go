package tradestation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
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

func (s *BrokerageService) GetOrdersByID(ctx context.Context, accountIDs, orderIDs []string) (*OrdersResponse, error) {
	if err := validateAccountIDs(accountIDs); err != nil {
		return nil, err
	}
	if err := validateOrderIDs(orderIDs); err != nil {
		return nil, err
	}
	path := "/v3/brokerage/accounts/" + strings.Join(accountIDs, ",") +
		"/orders/" + strings.Join(orderIDs, ",")
	var out OrdersResponse
	if err := s.client.doJSON(ctx, "GET", path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type historicalOrdersOpts struct {
	maxPages int
	pageSize int
}

type HistoricalOrdersOption func(*historicalOrdersOpts)

func WithMaxPages(n int) HistoricalOrdersOption {
	return func(o *historicalOrdersOpts) { o.maxPages = n }
}

func WithPageSize(n int) HistoricalOrdersOption {
	return func(o *historicalOrdersOpts) { o.pageSize = n }
}

func (s *BrokerageService) GetHistoricalOrders(
	ctx context.Context,
	accountIDs []string,
	since time.Time,
	opts ...HistoricalOrdersOption,
) (*OrdersResponse, error) {
	if err := validateAccountIDs(accountIDs); err != nil {
		return nil, err
	}
	return s.historicalOrdersLoop(
		ctx,
		"/v3/brokerage/accounts/"+strings.Join(accountIDs, ",")+"/historicalorders",
		since,
		opts,
	)
}

func (s *BrokerageService) historicalOrdersLoop(
	ctx context.Context,
	basePath string,
	since time.Time,
	opts []HistoricalOrdersOption,
) (*OrdersResponse, error) {
	cfg := historicalOrdersOpts{}
	for _, o := range opts {
		o(&cfg)
	}

	q := url.Values{}
	q.Set("since", since.UTC().Format("2006-01-02"))
	if cfg.pageSize > 0 {
		q.Set("pageSize", strconv.Itoa(cfg.pageSize))
	}

	agg := &OrdersResponse{}
	for pages := 0; ; pages++ {
		var page struct {
			Orders    []Order        `json:"Orders"`
			Errors    []AccountError `json:"Errors"`
			NextToken string         `json:"NextToken"`
		}
		if err := s.client.doJSON(ctx, "GET", basePath, q, nil, &page); err != nil {
			return nil, err
		}
		agg.Orders = append(agg.Orders, page.Orders...)
		agg.Errors = append(agg.Errors, page.Errors...)
		if page.NextToken == "" {
			break
		}
		if cfg.maxPages > 0 && pages+1 >= cfg.maxPages {
			break
		}
		q.Set("nextToken", page.NextToken)
	}
	return agg, nil
}
