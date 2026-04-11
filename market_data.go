package tradestation

type MarketDataService struct {
	client *Client
}

func (s *MarketDataService) GetBars(symbol string, interval string, barsBack int) ([]Bar, error) {
	panic("not implemented")
}

func (s *MarketDataService) GetQuote(symbols []string) ([]Quote, error) {
	panic("not implemented")
}

func (s *MarketDataService) GetOptionsChain(symbol string) (*OptionsChain, error) {
	panic("not implemented")
}

func (s *MarketDataService) StreamBars(symbol string, interval string) (<-chan Bar, error) {
	panic("not implemented")
}

func (s *MarketDataService) StreamQuotes(symbols []string) (<-chan Quote, error) {
	panic("not implemented")
}
