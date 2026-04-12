package tradestation

import (
	"context"
	"net/url"
	"strconv"
)

type MarketDataService struct {
	client *Client
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
	panic("not implemented") // Task 7
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
