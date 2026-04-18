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
	AccountID     string         `json:"AccountID"`
	AccountType   string         `json:"AccountType"`
	Alias         string         `json:"Alias,omitempty"`
	AltID         string         `json:"AltID,omitempty"`
	Currency      string         `json:"Currency"`
	Status        string         `json:"Status"`
	AccountDetail *AccountDetail `json:"AccountDetail,omitempty"`
}

type AccountDetail struct {
	CrossMarginAvailable       bool        `json:"CrossMarginAvailable"`
	DayTradingQualified        bool        `json:"DayTradingQualified"`
	EnrolledInRegTProgram      bool        `json:"EnrolledInRegTProgram"`
	IsStockLocateEligible      bool        `json:"IsStockLocateEligible"`
	OptionApprovalLevel        StringInt64 `json:"OptionApprovalLevel"`
	PatternDayTrader           bool        `json:"PatternDayTrader"`
	RequiresBuyingPowerWarning bool        `json:"RequiresBuyingPowerWarning"`
}

type Balance struct {
	AccountID        string           `json:"AccountID"`
	AccountType      string           `json:"AccountType"`
	CashBalance      StringFloat64    `json:"CashBalance"`
	BuyingPower      StringFloat64    `json:"BuyingPower"`
	Equity           StringFloat64    `json:"Equity"`
	MarketValue      StringFloat64    `json:"MarketValue"`
	TodaysProfitLoss StringFloat64    `json:"TodaysProfitLoss"`
	UnclearedDeposit StringFloat64    `json:"UnclearedDeposit"`
	CommissionFee    StringFloat64    `json:"CommissionFee"`
	BalanceDetail    *BalanceDetail   `json:"BalanceDetail,omitempty"`
	CurrencyDetails  []CurrencyDetail `json:"CurrencyDetails,omitempty"`
}

type BalanceDetail struct {
	CostOfPositions         StringFloat64 `json:"CostOfPositions"`
	DayTradeExcess          StringFloat64 `json:"DayTradeExcess"`
	DayTradeMargin          StringFloat64 `json:"DayTradeMargin"`
	DayTradeOpenOrderMargin StringFloat64 `json:"DayTradeOpenOrderMargin"`
	DayTrades               StringInt64   `json:"DayTrades"`
	InitialMargin           StringFloat64 `json:"InitialMargin"`
	MaintenanceMargin       StringFloat64 `json:"MaintenanceMargin"`
	MaintenanceRate         StringFloat64 `json:"MaintenanceRate"`
	MarginRequirement       StringFloat64 `json:"MarginRequirement"`
	UnrealizedProfitLoss    StringFloat64 `json:"UnrealizedProfitLoss"`
	UnsettledFunds          StringFloat64 `json:"UnsettledFunds"`
	OvernightBuyingPower    StringFloat64 `json:"OvernightBuyingPower"`
	OptionBuyingPower       StringFloat64 `json:"OptionBuyingPower"`
	OptionsMarketValue      StringFloat64 `json:"OptionsMarketValue"`
	StockBuyingPower        StringFloat64 `json:"StockBuyingPower"`
}

type CurrencyDetail struct {
	Currency                 string        `json:"Currency"`
	Commission               StringFloat64 `json:"Commission"`
	CashBalance              StringFloat64 `json:"CashBalance"`
	RealizedProfitLoss       StringFloat64 `json:"RealizedProfitLoss"`
	UnrealizedProfitLoss     StringFloat64 `json:"UnrealizedProfitLoss"`
	AccountMarginRequirement StringFloat64 `json:"AccountMarginRequirement,omitempty"`
	AccountOpenTradeEquity   StringFloat64 `json:"AccountOpenTradeEquity,omitempty"`
	AccountSecurities        StringFloat64 `json:"AccountSecurities,omitempty"`
	MarginRequirement        StringFloat64 `json:"MarginRequirement,omitempty"`
	NonTradeDebit            StringFloat64 `json:"NonTradeDebit,omitempty"`
	NonTradeNetBalance       StringFloat64 `json:"NonTradeNetBalance,omitempty"`
	OptionValue              StringFloat64 `json:"OptionValue,omitempty"`
	RealTimeAccountBalance   StringFloat64 `json:"RealTimeAccountBalance,omitempty"`
	RealTimeBuyingPower      StringFloat64 `json:"RealTimeBuyingPower,omitempty"`
	RealTimeEquity           StringFloat64 `json:"RealTimeEquity,omitempty"`
	RealTimeCostOfPositions  StringFloat64 `json:"RealTimeCostOfPositions,omitempty"`
	TodayRealTimeTradeEquity StringFloat64 `json:"TodayRealTimeTradeEquity,omitempty"`
	TradeEquity              StringFloat64 `json:"TradeEquity,omitempty"`
}

type BODBalance struct {
	AccountID       string              `json:"AccountID"`
	AccountType     string              `json:"AccountType"`
	BalanceDetail   *BODBalanceDetail   `json:"BalanceDetail,omitempty"`
	CurrencyDetails []BODCurrencyDetail `json:"CurrencyDetails,omitempty"`
}

type BODBalanceDetail struct {
	AccountBalance                  StringFloat64 `json:"AccountBalance"`
	CashAvailableToWithdraw         StringFloat64 `json:"CashAvailableToWithdraw"`
	DayTrades                       StringInt64   `json:"DayTrades"`
	DayTradingMarginableBuyingPower StringFloat64 `json:"DayTradingMarginableBuyingPower"`
	Equity                          StringFloat64 `json:"Equity"`
	NetCash                         StringFloat64 `json:"NetCash"`
	OvernightBuyingPower            StringFloat64 `json:"OvernightBuyingPower"`
	OptionBuyingPower               StringFloat64 `json:"OptionBuyingPower"`
	OptionValue                     StringFloat64 `json:"OptionValue"`
	RegTCall                        StringFloat64 `json:"RegTCall"`
	RegTEquity                      StringFloat64 `json:"RegTEquity"`
	RegTEquityPercentage            StringFloat64 `json:"RegTEquityPercentage"`
	RegTMargin                      StringFloat64 `json:"RegTMargin"`
	RegTMarginEquity                StringFloat64 `json:"RegTMarginEquity"`
	RegTMarginEquityPercentage      StringFloat64 `json:"RegTMarginEquityPercentage"`
	RequiredMargin                  StringFloat64 `json:"RequiredMargin"`
	UnsettledFunds                  StringFloat64 `json:"UnsettledFunds"`
	DayTradingBuyingPower           StringFloat64 `json:"DayTradingBuyingPower"`
}

type BODCurrencyDetail struct {
	Currency                 string        `json:"Currency"`
	AccountMarginRequirement StringFloat64 `json:"AccountMarginRequirement"`
	AccountConversionRate    StringFloat64 `json:"AccountConversionRate"`
	BODCashBalance           StringFloat64 `json:"BODCashBalance"`
	BODOpenTradeEquity       StringFloat64 `json:"BODOpenTradeEquity"`
	BODSecurities            StringFloat64 `json:"BODSecurities"`
	MarginRequirement        StringFloat64 `json:"MarginRequirement"`
	NonTradeDebit            StringFloat64 `json:"NonTradeDebit"`
	NonTradeNetBalance       StringFloat64 `json:"NonTradeNetBalance"`
	OptionValue              StringFloat64 `json:"OptionValue"`
}

type Position struct {
	PositionID                  string        `json:"PositionID"`
	AccountID                   string        `json:"AccountID"`
	Symbol                      string        `json:"Symbol"`
	AssetType                   string        `json:"AssetType"`
	Quantity                    StringInt64   `json:"Quantity"`
	AveragePrice                StringFloat64 `json:"AveragePrice"`
	Last                        StringFloat64 `json:"Last"`
	Bid                         StringFloat64 `json:"Bid"`
	Ask                         StringFloat64 `json:"Ask"`
	MarketValue                 StringFloat64 `json:"MarketValue"`
	UnrealizedProfitLoss        StringFloat64 `json:"UnrealizedProfitLoss"`
	UnrealizedProfitLossPercent StringFloat64 `json:"UnrealizedProfitLossPercent"`
	UnrealizedProfitLossQty     StringFloat64 `json:"UnrealizedProfitLossQty"`
	TodaysProfitLoss            StringFloat64 `json:"TodaysProfitLoss"`
	TotalCost                   StringFloat64 `json:"TotalCost"`
	MarkToMarketPrice           StringFloat64 `json:"MarkToMarketPrice"`
	InitialRequirement          StringFloat64 `json:"InitialRequirement"`
	MaintenanceMargin           StringFloat64 `json:"MaintenanceMargin"`
	ConversionRate              StringFloat64 `json:"ConversionRate"`
	DayTradeRequirement         StringFloat64 `json:"DayTradeRequirement"`
	LongShort                   string        `json:"LongShort"`
	Timestamp                   time.Time     `json:"Timestamp"`
}

// Order Execution types

type Order struct {
	OrderID           string             `json:"OrderID"`
	AccountID         string             `json:"AccountID"`
	Status            string             `json:"Status"`
	StatusDescription string             `json:"StatusDescription"`
	OrderType         string             `json:"OrderType"`
	Duration          string             `json:"Duration"`
	GoodTillDate      string             `json:"GoodTillDate,omitempty"`
	LimitPrice        StringFloat64      `json:"LimitPrice,omitempty"`
	StopPrice         StringFloat64      `json:"StopPrice,omitempty"`
	FilledPrice       StringFloat64      `json:"FilledPrice,omitempty"`
	CommissionFee     StringFloat64      `json:"CommissionFee"`
	Currency          string             `json:"Currency"`
	OpenedDateTime    time.Time          `json:"OpenedDateTime"`
	ClosedDateTime    time.Time          `json:"ClosedDateTime,omitempty"`
	Legs              []OrderLeg         `json:"Legs,omitempty"`
	Routes            []OrderRoute       `json:"Routes,omitempty"`
	AdvancedOptions   string             `json:"AdvancedOptions,omitempty"`
	ConditionalOrders []ConditionalOrder `json:"ConditionalOrders,omitempty"`
}

type OrderLeg struct {
	AssetType         string        `json:"AssetType"`
	BuyOrSell         string        `json:"BuyOrSell"`
	ExecQuantity      StringInt64   `json:"ExecQuantity"`
	ExecutionPrice    StringFloat64 `json:"ExecutionPrice"`
	ExpirationDate    string        `json:"ExpirationDate,omitempty"`
	OpenOrClose       string        `json:"OpenOrClose,omitempty"`
	OptionType        string        `json:"OptionType,omitempty"`
	QuantityOrdered   StringInt64   `json:"QuantityOrdered"`
	QuantityRemaining StringInt64   `json:"QuantityRemaining"`
	StrikePrice       StringFloat64 `json:"StrikePrice,omitempty"`
	Symbol            string        `json:"Symbol"`
	Underlying        string        `json:"Underlying,omitempty"`
}

type OrderRoute struct {
	Name       string   `json:"Name"`
	ID         string   `json:"Id"`
	AssetTypes []string `json:"AssetTypes"`
}

type ConditionalOrder struct {
	OrderID      string `json:"OrderID"`
	Relationship string `json:"Relationship"`
}

// OrderRequest stays as-is (used by OrderService stubs, out of scope for this branch).
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

// Brokerage response wrappers — carry both data and partial per-account errors.

type AccountError struct {
	AccountID string `json:"AccountID"`
	ErrorCode string `json:"Error"`
	Message   string `json:"Message"`
}

type BalancesResponse struct {
	Balances []Balance      `json:"Balances"`
	Errors   []AccountError `json:"Errors,omitempty"`
}

type BalancesBODResponse struct {
	BODBalances []BODBalance   `json:"BODBalances"`
	Errors      []AccountError `json:"Errors,omitempty"`
}

type PositionsResponse struct {
	Positions []Position     `json:"Positions"`
	Errors    []AccountError `json:"Errors,omitempty"`
}

type OrdersResponse struct {
	Orders []Order        `json:"Orders"`
	Errors []AccountError `json:"Errors,omitempty"`
}
