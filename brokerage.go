package tradestation

import "time"

type BrokerageService struct {
	client *Client
}

func (s *BrokerageService) GetAccounts() ([]Account, error) {
	panic("not implemented")
}

func (s *BrokerageService) GetPositions(accountID string) ([]Position, error) {
	panic("not implemented")
}

func (s *BrokerageService) GetBalances(accountID string) (*Balance, error) {
	panic("not implemented")
}

func (s *BrokerageService) GetOrders(accountID string) ([]Order, error) {
	panic("not implemented")
}

func (s *BrokerageService) GetHistoricalOrders(accountID string, since time.Time) ([]Order, error) {
	panic("not implemented")
}
