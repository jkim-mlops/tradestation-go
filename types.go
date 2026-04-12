package tradestation

import (
	"encoding/json"
	"strconv"
	"time"
)

// StringFloat64 unmarshals from both JSON strings ("1.5") and numbers (1.5).
type StringFloat64 float64

func (f *StringFloat64) UnmarshalJSON(data []byte) error {
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = StringFloat64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = StringFloat64(n)
	return nil
}

// StringInt64 unmarshals from both JSON strings ("100") and numbers (100).
type StringInt64 int64

func (i *StringInt64) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*i = StringInt64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*i = StringInt64(n)
	return nil
}

// MarketData types

type Bar struct {
	High            StringFloat64 `json:"High"`
	Low             StringFloat64 `json:"Low"`
	Open            StringFloat64 `json:"Open"`
	Close           StringFloat64 `json:"Close"`
	TimeStamp       time.Time     `json:"TimeStamp"`
	TotalVolume     StringInt64   `json:"TotalVolume"`
	DownTicks       StringInt64   `json:"DownTicks"`
	DownVolume      StringInt64   `json:"DownVolume"`
	OpenInterest    StringInt64   `json:"OpenInterest"`
	IsRealtime      bool          `json:"IsRealtime"`
	TotalTicks      StringInt64   `json:"TotalTicks"`
	UnchangedTicks  StringInt64   `json:"UnchangedTicks"`
	UnchangedVolume StringInt64   `json:"UnchangedVolume"`
	UpTicks         StringInt64   `json:"UpTicks"`
	UpVolume        StringInt64   `json:"UpVolume"`
	Epoch           StringInt64   `json:"Epoch"`
	BarStatus       string        `json:"BarStatus"`
}

type Quote struct {
	Symbol            string        `json:"Symbol"`
	Ask               StringFloat64 `json:"Ask"`
	AskSize           StringInt64   `json:"AskSize"`
	Bid               StringFloat64 `json:"Bid"`
	BidSize           StringInt64   `json:"BidSize"`
	Last              StringFloat64 `json:"Last"`
	LastSize          StringInt64   `json:"LastSize"`
	Volume            StringInt64   `json:"Volume"`
	Close             StringFloat64 `json:"Close"`
	High52Week        StringFloat64 `json:"High52Week"`
	Low52Week         StringFloat64 `json:"Low52Week"`
	DailyOpenInterest StringInt64   `json:"DailyOpenInterest"`
}

type OptionsChain struct {
	Expirations []OptionsExpiration `json:"Expirations"`
}

type OptionsExpiration struct {
	Date    string          `json:"Date"`
	Strikes []OptionsStrike `json:"Strikes"`
}

type OptionsStrike struct {
	StrikePrice StringFloat64 `json:"StrikePrice"`
	Call        *OptionsLeg   `json:"Call"`
	Put         *OptionsLeg   `json:"Put"`
}

type OptionsLeg struct {
	Symbol            string        `json:"Symbol"`
	Ask               StringFloat64 `json:"Ask"`
	Bid               StringFloat64 `json:"Bid"`
	Last              StringFloat64 `json:"Last"`
	Volume            StringInt64   `json:"Volume"`
	OpenInterest      StringInt64   `json:"OpenInterest"`
	ImpliedVolatility StringFloat64 `json:"ImpliedVolatility"`
	Delta             StringFloat64 `json:"Delta"`
	Gamma             StringFloat64 `json:"Gamma"`
	Theta             StringFloat64 `json:"Theta"`
	Vega              StringFloat64 `json:"Vega"`
}

// Brokerage types

type Account struct {
	AccountID   string `json:"AccountID"`
	AccountType string `json:"AccountType"`
	Status      string `json:"Status"`
}

type Position struct {
	AccountID         string        `json:"AccountID"`
	Symbol            string        `json:"Symbol"`
	Quantity          StringInt64   `json:"Quantity"`
	AveragePrice      StringFloat64 `json:"AveragePrice"`
	Last              StringFloat64 `json:"Last"`
	MarketValue       StringFloat64 `json:"MarketValue"`
	UnrealizedPnL     StringFloat64 `json:"UnrealizedPL"`
	LongShort         string        `json:"LongShort"`
	AssetType         string        `json:"AssetType"`
	ConversionRate    StringFloat64 `json:"ConversionRate"`
	InitialMargin     StringFloat64 `json:"InitialMargin"`
	MaintenanceMargin StringFloat64 `json:"MaintenanceMargin"`
	TotalCost         StringFloat64 `json:"TotalCost"`
	Timestamp         string        `json:"Timestamp"`
}

type Balance struct {
	AccountID     string        `json:"AccountID"`
	CashBalance   StringFloat64 `json:"CashBalance"`
	BuyingPower   StringFloat64 `json:"BuyingPower"`
	Equity        StringFloat64 `json:"Equity"`
	MarketValue   StringFloat64 `json:"MarketValue"`
	RealizedPnL   StringFloat64 `json:"RealizedPL"`
	UnrealizedPnL StringFloat64 `json:"UnrealizedPL"`
}

// Order Execution types

type Order struct {
	OrderID        string        `json:"OrderID"`
	AccountID      string        `json:"AccountID"`
	Symbol         string        `json:"Symbol"`
	Quantity       StringInt64   `json:"Quantity"`
	FilledQuantity StringInt64   `json:"FilledQuantity"`
	OrderType      string        `json:"OrderType"`
	LimitPrice     StringFloat64 `json:"LimitPrice"`
	StopPrice      StringFloat64 `json:"StopPrice"`
	Side           string        `json:"Side"`
	Status         string        `json:"Status"`
	Duration       string        `json:"Duration"`
	FilledPrice    StringFloat64 `json:"FilledPrice"`
	OpenedDateTime string        `json:"OpenedDateTime"`
	ClosedDateTime string        `json:"ClosedDateTime"`
	TimeInForce    string        `json:"TimeInForce"`
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
