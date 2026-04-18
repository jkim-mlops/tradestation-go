package tradestation

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type MarketDataService struct {
	client *Client
}

// MarketData returns a MarketDataService bound to this client.
func (c *Client) MarketData() *MarketDataService {
	return &MarketDataService{client: c}
}

type BarUnit string

const (
	BarUnitMinute  BarUnit = "Minute"
	BarUnitDaily   BarUnit = "Daily"
	BarUnitWeekly  BarUnit = "Weekly"
	BarUnitMonthly BarUnit = "Monthly"
)

type GetBarsParams struct {
	Interval  int // 1-1440; must be 1 for Daily/Weekly/Monthly
	Unit      BarUnit
	BarsBack  int    // optional; when 0, uses StartDate
	StartDate string // optional ISO8601
}

func (s *MarketDataService) GetBars(ctx context.Context, symbol string, params GetBarsParams) ([]Bar, error) {
	q := url.Values{}
	q.Set("interval", strconv.Itoa(params.Interval))
	q.Set("unit", string(params.Unit))
	if params.BarsBack > 0 {
		q.Set("barsback", strconv.Itoa(params.BarsBack))
	}
	if params.StartDate != "" {
		q.Set("startdate", params.StartDate)
	}

	var resp struct {
		Bars []Bar `json:"Bars"`
	}
	path := "/v3/marketdata/barcharts/" + url.PathEscape(symbol)
	if err := s.client.doJSON(ctx, "GET", path, q, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Bars, nil
}

func (s *MarketDataService) GetQuote(ctx context.Context, symbols []string) ([]Quote, error) {
	if len(symbols) == 0 {
		return nil, errors.New("tradestation: GetQuote requires at least one symbol")
	}
	if len(symbols) > 50 {
		return nil, errors.New("tradestation: GetQuote supports at most 50 symbols per request")
	}

	path := "/v3/marketdata/quotes/" + strings.Join(symbols, ",")

	var resp struct {
		Quotes []Quote `json:"Quotes"`
	}
	if err := s.client.doJSON(ctx, "GET", path, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Quotes, nil
}

func (s *MarketDataService) GetOptionsChain(symbol string) (*OptionsChain, error) {
	panic("not implemented") // Phase 2D: streaming-only in the spec
}

func (s *MarketDataService) StreamBars(symbol string, interval string) (<-chan Bar, error) {
	panic("not implemented")
}

func (s *MarketDataService) StreamQuotes(symbols []string) (<-chan Quote, error) {
	panic("not implemented")
}
