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
		case ev.Data != nil:
			if ev.Data.Symbol != "AAPL" || ev.Data.Last != 150.5 {
				t.Errorf("quote decoded wrong: %+v", *ev.Data)
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

func TestStreamQuotes_HeartbeatsFiltered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := w.(http.Flusher)
		w.Write([]byte(`{"Symbol":"AAPL","Last":150.5}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"Heartbeat":1,"Timestamp":"2026-04-19T16:52:56Z"}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"Heartbeat":2,"Timestamp":"2026-04-19T16:53:01Z"}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"Symbol":"MSFT","Last":300.1}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	events, err := svc.StreamQuotes(context.Background(), []string{"AAPL", "MSFT"}, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamQuotes: %v", err)
	}

	var got []Quote
	for ev := range events {
		if ev.Err != nil {
			t.Fatalf("unexpected error: %v", ev.Err)
		}
		if ev.Data != nil {
			got = append(got, *ev.Data)
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d quotes, want 2 (heartbeats should be filtered): %+v", len(got), got)
	}
	if got[0].Symbol != "AAPL" || got[1].Symbol != "MSFT" {
		t.Errorf("symbols = %q,%q", got[0].Symbol, got[1].Symbol)
	}
}

func TestStreamBars_HappyPath(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		f := w.(http.Flusher)
		w.Write([]byte(`{"Open":"100","High":"101","Low":"99","Close":"100.5","TotalVolume":"1000","TimeStamp":"2026-04-19T15:00:00Z"}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"StreamStatus":"EndSnapshot"}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	events, err := svc.StreamBars(
		context.Background(),
		"AAPL",
		StreamBarsParams{Interval: 1, Unit: BarUnitMinute, BarsBack: 5},
		WithoutReconnect(),
	)
	if err != nil {
		t.Fatalf("StreamBars: %v", err)
	}

	var gotBar bool
	for ev := range events {
		if ev.Err != nil {
			t.Fatalf("err: %v", ev.Err)
		}
		if ev.Data != nil && ev.Data.Close == 100.5 {
			gotBar = true
		}
	}
	if gotPath != "/v3/marketdata/stream/barcharts/AAPL" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "barsback=5&interval=1&unit=Minute" {
		t.Errorf("query = %q", gotQuery)
	}
	if !gotBar {
		t.Error("no bar event received")
	}
}

func TestStreamBars_SessionTemplate(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.(http.Flusher).Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &MarketDataService{client: c}

	events, err := svc.StreamBars(
		context.Background(),
		"SPY",
		StreamBarsParams{Interval: 5, Unit: BarUnitMinute, SessionTemplate: "USEQPreAndPost"},
		WithoutReconnect(),
	)
	if err != nil {
		t.Fatalf("StreamBars: %v", err)
	}
	for range events {
	}
	if gotQuery != "interval=5&sessiontemplate=USEQPreAndPost&unit=Minute" {
		t.Errorf("query = %q", gotQuery)
	}
}

func TestStreamBars_ValidationRejectsEmptySymbol(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	_, err := svc.StreamBars(context.Background(), "", StreamBarsParams{Interval: 1, Unit: BarUnitMinute})
	if err == nil {
		t.Error("want error for empty symbol")
	}
}

func TestStreamBars_ValidationRejectsBadInterval(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	_, err := svc.StreamBars(context.Background(), "AAPL", StreamBarsParams{Interval: 0, Unit: BarUnitMinute})
	if err == nil {
		t.Error("want error for interval=0")
	}
	_, err = svc.StreamBars(context.Background(), "AAPL", StreamBarsParams{Interval: 1441, Unit: BarUnitMinute})
	if err == nil {
		t.Error("want error for interval=1441")
	}
}

func TestStreamBars_ValidationRejectsNonMinuteIntervalGtOne(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &MarketDataService{client: c}
	_, err := svc.StreamBars(context.Background(), "AAPL", StreamBarsParams{Interval: 2, Unit: BarUnitDaily})
	if err == nil {
		t.Error("want error for Daily with interval>1")
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
