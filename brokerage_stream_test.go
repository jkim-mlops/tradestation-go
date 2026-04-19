package tradestation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamOrders_HappyPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		f := w.(http.Flusher)
		w.Write([]byte(`{"OrderID":"o1","AccountID":"123","Status":"Open"}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"StreamStatus":"EndSnapshot"}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	events, err := svc.StreamOrders(context.Background(), []string{"123", "456"}, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamOrders: %v", err)
	}

	var gotOrder, gotSnapshot bool
	for ev := range events {
		switch {
		case ev.Err != nil:
			t.Fatalf("err: %v", ev.Err)
		case ev.Data != nil:
			if ev.Data.OrderID != "o1" || ev.Data.AccountID != "123" {
				t.Errorf("order decoded wrong: %+v", *ev.Data)
			}
			gotOrder = true
		case ev.Status == StreamStatusEndSnapshot:
			gotSnapshot = true
		}
	}
	if gotPath != "/v3/brokerage/stream/accounts/123,456/orders" {
		t.Errorf("path = %q", gotPath)
	}
	if !gotOrder {
		t.Error("no order event received")
	}
	if !gotSnapshot {
		t.Error("no EndSnapshot received")
	}
}

func TestStreamOrders_ValidationRejectsEmpty(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &BrokerageService{client: c}
	if _, err := svc.StreamOrders(context.Background(), nil); err == nil {
		t.Error("want error for empty accountIDs")
	}
}

func TestStreamOrders_ValidationRejectsTooMany(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &BrokerageService{client: c}
	ids := make([]string, 26)
	for i := range ids {
		ids[i] = "a"
	}
	if _, err := svc.StreamOrders(context.Background(), ids); err == nil {
		t.Error("want error for >25 accountIDs")
	}
}

func TestStreamOrdersByID_HappyPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		f := w.(http.Flusher)
		w.Write([]byte(`{"OrderID":"o1","AccountID":"123"}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"StreamStatus":"EndSnapshot"}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	events, err := svc.StreamOrdersByID(
		context.Background(),
		[]string{"123"},
		[]string{"o1", "o2"},
		WithoutReconnect(),
	)
	if err != nil {
		t.Fatalf("StreamOrdersByID: %v", err)
	}
	for range events {
	}
	if gotPath != "/v3/brokerage/stream/accounts/123/orders/o1,o2" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestStreamOrdersByID_ValidationRejectsEmptyOrderIDs(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &BrokerageService{client: c}
	if _, err := svc.StreamOrdersByID(context.Background(), []string{"123"}, nil); err == nil {
		t.Error("want error for empty orderIDs")
	}
}

func TestStreamPositions_HappyPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		f := w.(http.Flusher)
		w.Write([]byte(`{"AccountID":"123","Symbol":"AAPL","Quantity":"10"}` + "\n"))
		f.Flush()
		w.Write([]byte(`{"StreamStatus":"EndSnapshot"}` + "\n"))
		f.Flush()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL
	svc := &BrokerageService{client: c}

	events, err := svc.StreamPositions(context.Background(), []string{"123"}, WithoutReconnect())
	if err != nil {
		t.Fatalf("StreamPositions: %v", err)
	}
	var gotPosition bool
	for ev := range events {
		if ev.Err != nil {
			t.Fatalf("err: %v", ev.Err)
		}
		if ev.Data != nil && ev.Data.Symbol == "AAPL" {
			gotPosition = true
		}
	}
	if gotPath != "/v3/brokerage/stream/accounts/123/positions" {
		t.Errorf("path = %q", gotPath)
	}
	if !gotPosition {
		t.Error("no position event received")
	}
}

func TestStreamPositions_ValidationRejectsEmpty(t *testing.T) {
	c := NewClient(Test, "id", "secret", "refresh")
	svc := &BrokerageService{client: c}
	if _, err := svc.StreamPositions(context.Background(), nil); err == nil {
		t.Error("want error for empty accountIDs")
	}
}
