package tradestation

import "fmt"

// APIError is returned when TradeStation responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Message    string
	RawBody    []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("tradestation: %d %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("tradestation: %d", e.StatusCode)
}
