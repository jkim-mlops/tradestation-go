package tradestation

import (
	"errors"
	"testing"
)

func TestAPIError_Error_WithMessage(t *testing.T) {
	e := &APIError{StatusCode: 429, Message: "rate limited"}
	got := e.Error()
	want := "tradestation: 429 rate limited"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIError_Error_WithoutMessage(t *testing.T) {
	e := &APIError{StatusCode: 500}
	got := e.Error()
	want := "tradestation: 500"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIError_ErrorsAs(t *testing.T) {
	var err error = &APIError{StatusCode: 404, Message: "not found"}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As failed")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}
