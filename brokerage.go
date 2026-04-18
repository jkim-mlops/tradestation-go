package tradestation

import (
	"errors"
	"fmt"
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
