package tradestation

type OrderService struct {
	client *Client
}

func (s *OrderService) PlaceOrder(req OrderRequest) (*Order, error) {
	panic("not implemented")
}

func (s *OrderService) ReplaceOrder(orderID string, req OrderRequest) (*Order, error) {
	panic("not implemented")
}

func (s *OrderService) CancelOrder(orderID string) error {
	panic("not implemented")
}
