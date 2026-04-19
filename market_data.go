package tradestation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

// QuoteEvent is a single event on a quote stream. Exactly one of Quote, Status,
// or Err is populated per event. The event channel closes after a terminal
// event (Err populated, or clean termination).
type QuoteEvent struct {
	Quote  *Quote
	Status StreamStatus
	Err    error
}

func (s *MarketDataService) StreamQuotes(
	ctx context.Context,
	symbols []string,
	opts ...StreamOption,
) (<-chan QuoteEvent, error) {
	if len(symbols) == 0 {
		return nil, errors.New("tradestation: StreamQuotes requires at least one symbol")
	}
	if len(symbols) > 50 {
		return nil, errors.New("tradestation: StreamQuotes supports at most 50 symbols per request")
	}

	cfg := defaultStreamOpts()
	for _, o := range opts {
		o(&cfg)
	}

	path := "/v3/marketdata/stream/quotes/" + strings.Join(symbols, ",")
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", s.client.apiBase+path, nil)
	}

	// Synchronous first attempt so initial connect errors (4xx/5xx) return here
	// rather than through the channel. authTransport handles 401 refresh-and-retry.
	req, err := openReq(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	raw := make(chan streamEvent)
	events := make(chan QuoteEvent)

	go s.client.runStreamFromResp(ctx, resp, openReq, raw, cfg)
	go pumpQuoteEvents(raw, events)

	return events, nil
}

func pumpQuoteEvents(in <-chan streamEvent, out chan<- QuoteEvent) {
	defer close(out)
	for ev := range in {
		switch {
		case ev.Err != nil:
			out <- QuoteEvent{Err: ev.Err}
		case ev.Status != "":
			out <- QuoteEvent{Status: ev.Status}
		default:
			var q Quote
			if err := json.Unmarshal(ev.Raw, &q); err != nil {
				out <- QuoteEvent{Err: fmt.Errorf("tradestation: decode quote: %w", err)}
				continue
			}
			out <- QuoteEvent{Quote: &q}
		}
	}
}
