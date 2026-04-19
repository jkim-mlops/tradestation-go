package tradestation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetBars_RequestShape(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Bars":[{"Open":1,"High":2,"Low":0.5,"Close":1.5,"TotalVolume":100}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	bars, err := svc.GetBars(context.Background(), "AAPL", GetBarsParams{
		Interval: 1, Unit: BarUnitDaily, BarsBack: 10,
	})
	if err != nil {
		t.Fatalf("GetBars: %v", err)
	}
	if gotPath != "/v3/marketdata/barcharts/AAPL" {
		t.Errorf("path = %q", gotPath)
	}
	// url.Values.Encode sorts keys alphabetically.
	if gotQuery != "barsback=10&interval=1&unit=Daily" {
		t.Errorf("query = %q", gotQuery)
	}
	if len(bars) != 1 || bars[0].Open != 1 {
		t.Errorf("bars decoded wrong: %+v", bars)
	}
}

func TestGetBars_WithStartDate(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"Bars":[]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	_, err := svc.GetBars(context.Background(), "MSFT", GetBarsParams{
		Interval: 1, Unit: BarUnitMinute, StartDate: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("GetBars: %v", err)
	}
	want := "interval=1&startdate=2026-01-01T00%3A00%3A00Z&unit=Minute"
	if gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
}

func TestGetQuote_RequestShape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"Quotes":[{"Symbol":"AAPL","Last":150.5},{"Symbol":"MSFT","Last":300.1}]}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	quotes, err := svc.GetQuote(context.Background(), []string{"AAPL", "MSFT"})
	if err != nil {
		t.Fatalf("GetQuote: %v", err)
	}
	if gotPath != "/v3/marketdata/quotes/AAPL,MSFT" {
		t.Errorf("path = %q", gotPath)
	}
	if len(quotes) != 2 || quotes[0].Symbol != "AAPL" || quotes[1].Last != 300.1 {
		t.Errorf("quotes decoded wrong: %+v", quotes)
	}
}

func TestGetQuote_EmptyRejected(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	_, err := svc.GetQuote(context.Background(), nil)
	if err == nil {
		t.Error("want error for empty symbols")
	}
}

func TestGetQuote_TooManyRejected(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	syms := make([]string, 51)
	for i := range syms {
		syms[i] = "X"
	}
	_, err := svc.GetQuote(context.Background(), syms)
	if err == nil {
		t.Error("want error for >50 symbols")
	}
}

func TestStreamQuotes_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.tradestation.streams.v2+json")
		f := w.(http.Flusher)
		w.Write([]byte(`{"Symbol":"AAPL","Last":150.5}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"StreamStatus":"EndSnapshot"}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := svc.StreamQuotes(ctx, []string{"AAPL"}, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamQuotes: %v", err)
	}

	var quoteSeen, snapshotSeen bool
	for ev := range events {
		switch {
		case ev.Err != nil:
			t.Fatalf("unexpected error: %v", ev.Err)
		case ev.Quote != nil:
			if ev.Quote.Symbol != "AAPL" || ev.Quote.Last != 150.5 {
				t.Errorf("quote decoded wrong: %+v", *ev.Quote)
			}
			quoteSeen = true
		case ev.Status != "":
			if ev.Status == StreamStatusEndSnapshot {
				snapshotSeen = true
			}
		}
	}
	if !quoteSeen || !snapshotSeen {
		t.Errorf("quoteSeen=%v snapshotSeen=%v", quoteSeen, snapshotSeen)
	}
}

func TestStreamQuotes_ErrorMessageTerminates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := w.(http.Flusher)
		w.Write([]byte(`{"Error":"DualLogon","Message":"another client"}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	events, err := svc.StreamQuotes(context.Background(), []string{"AAPL"})
	if err != nil {
		t.Fatalf("StreamQuotes: %v", err)
	}

	var gotErr error
	for ev := range events {
		if ev.Err != nil {
			gotErr = ev.Err
		}
	}
	var se *StreamError
	if !errors.As(gotErr, &se) {
		t.Fatalf("want *StreamError, got %v", gotErr)
	}
	if se.Code != "DualLogon" {
		t.Errorf("Code = %q", se.Code)
	}
}

func TestStreamQuotes_EmptySymbolsRejected(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	if _, err := svc.StreamQuotes(context.Background(), nil); err == nil {
		t.Error("want error for empty symbols")
	}
}

func TestStreamQuotes_TooManySymbolsRejected(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	syms := make([]string, 51)
	for i := range syms {
		syms[i] = "X"
	}
	if _, err := svc.StreamQuotes(context.Background(), syms); err == nil {
		t.Error("want error for >50 symbols")
	}
}

func TestStreamQuotes_ConnectErrorReturnsImmediately(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"Error":"BadRequest","Message":"bad"}`))
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	_, err := svc.StreamQuotes(context.Background(), []string{"AAPL"})
	if err == nil {
		t.Fatal("want error")
	}
	var ae *APIError
	if !errors.As(err, &ae) || ae.StatusCode != http.StatusBadRequest {
		t.Errorf("want *APIError 400, got %v", err)
	}
}
