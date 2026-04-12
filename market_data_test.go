package tradestation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
