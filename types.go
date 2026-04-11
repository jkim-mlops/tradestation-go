package tradestation

import "time"

// MarketData types

type Bar struct {
	High            float64   `json:"High"`
	Low             float64   `json:"Low"`
	Open            float64   `json:"Open"`
	Close           float64   `json:"Close"`
	TimeStamp       time.Time `json:"TimeStamp"`
	TotalVolume     int64     `json:"TotalVolume"`
	DownTicks       int64     `json:"DownTicks"`
	DownVolume      int64     `json:"DownVolume"`
	OpenInterest    int64     `json:"OpenInterest"`
	IsRealtime      bool      `json:"IsRealtime"`
	TotalTicks      int64     `json:"TotalTicks"`
	UnchangedTicks  int64     `json:"UnchangedTicks"`
	UnchangedVolume int64     `json:"UnchangedVolume"`
	UpTicks         int64     `json:"UpTicks"`
	UpVolume        int64     `json:"UpVolume"`
	Epoch           int64     `json:"Epoch"`
	BarStatus       string    `json:"BarStatus"`
}

type Quote struct {
	Symbol            string  `json:"Symbol"`
	Ask               float64 `json:"Ask"`
	AskSize           int64   `json:"AskSize"`
	Bid               float64 `json:"Bid"`
	BidSize           int64   `json:"BidSize"`
	Last              float64 `json:"Last"`
	LastSize          int64   `json:"LastSize"`
	Volume            int64   `json:"Volume"`
	Close             float64 `json:"Close"`
	High52Week        float64 `json:"High52Week"`
	Low52Week         float64 `json:"Low52Week"`
	DailyOpenInterest int64   `json:"DailyOpenInterest"`
}

type OptionsChain struct {
	Expirations []OptionsExpiration `json:"Expirations"`
}

type OptionsExpiration struct {
	Date    string          `json:"Date"`
	Strikes []OptionsStrike `json:"Strikes"`
}

type OptionsStrike struct {
	StrikePrice float64     `json:"StrikePrice"`
	Call        *OptionsLeg `json:"Call"`
	Put         *OptionsLeg `json:"Put"`
}

type OptionsLeg struct {
	Symbol            string  `json:"Symbol"`
	Ask               float64 `json:"Ask"`
	Bid               float64 `json:"Bid"`
	Last              float64 `json:"Last"`
	Volume            int64   `json:"Volume"`
	OpenInterest      int64   `json:"OpenInterest"`
	ImpliedVolatility float64 `json:"ImpliedVolatility"`
	Delta             float64 `json:"Delta"`
	Gamma             float64 `json:"Gamma"`
	Theta             float64 `json:"Theta"`
	Vega              float64 `json:"Vega"`
}

// Brokerage types

type Account struct {
	AccountID   string `json:"AccountID"`
	AccountType string `json:"AccountType"`
	Status      string `json:"Status"`
}

type Position struct {
	AccountID         string  `json:"AccountID"`
	Symbol            string  `json:"Symbol"`
	Quantity          int64   `json:"Quantity"`
	AveragePrice      float64 `json:"AveragePrice"`
	Last              float64 `json:"Last"`
	MarketValue       float64 `json:"MarketValue"`
	UnrealizedPnL     float64 `json:"UnrealizedPL"`
	LongShort         string  `json:"LongShort"`
	AssetType         string  `json:"AssetType"`
	ConversionRate    float64 `json:"ConversionRate"`
	InitialMargin     float64 `json:"InitialMargin"`
	MaintenanceMargin float64 `json:"MaintenanceMargin"`
	TotalCost         float64 `json:"TotalCost"`
	Timestamp         string  `json:"Timestamp"`
}

type Balance struct {
	AccountID     string  `json:"AccountID"`
	CashBalance   float64 `json:"CashBalance"`
	BuyingPower   float64 `json:"BuyingPower"`
	Equity        float64 `json:"Equity"`
	MarketValue   float64 `json:"MarketValue"`
	RealizedPnL   float64 `json:"RealizedPL"`
	UnrealizedPnL float64 `json:"UnrealizedPL"`
}

// Order Execution types

type Order struct {
	OrderID        string  `json:"OrderID"`
	AccountID      string  `json:"AccountID"`
	Symbol         string  `json:"Symbol"`
	Quantity       int64   `json:"Quantity"`
	FilledQuantity int64   `json:"FilledQuantity"`
	OrderType      string  `json:"OrderType"`
	LimitPrice     float64 `json:"LimitPrice"`
	StopPrice      float64 `json:"StopPrice"`
	Side           string  `json:"Side"`
	Status         string  `json:"Status"`
	Duration       string  `json:"Duration"`
	FilledPrice    float64 `json:"FilledPrice"`
	OpenedDateTime string  `json:"OpenedDateTime"`
	ClosedDateTime string  `json:"ClosedDateTime"`
	TimeInForce    string  `json:"TimeInForce"`
}

type OrderRequest struct {
	AccountID   string  `json:"AccountID"`
	Symbol      string  `json:"Symbol"`
	Quantity    int64   `json:"Quantity"`
	OrderType   string  `json:"OrderType"`
	LimitPrice  float64 `json:"LimitPrice,omitempty"`
	StopPrice   float64 `json:"StopPrice,omitempty"`
	TradeAction string  `json:"TradeAction"`
	Duration    string  `json:"Duration"`
	Route       string  `json:"Route,omitempty"`
}
