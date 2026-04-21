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

// MarshalJSON emits a JSON-encoded string ("10.5"), matching TradeStation's
// wire format for numeric fields in request bodies.
func (f StringFloat64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.AppendQuote(nil, strconv.FormatFloat(float64(f), 'f', -1, 64))), nil
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

// MarshalJSON emits a JSON-encoded string ("100"), matching TradeStation's
// wire format for numeric fields in request bodies.
func (i StringInt64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.AppendQuote(nil, strconv.FormatInt(int64(i), 10))), nil
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

// OrderRequest is the body of POST /orderexecution/orders and POST /orderconfirm.
// Required fields: AccountID, Symbol, Quantity, OrderType, TradeAction,
// TimeInForce.Duration. OrderType determines which price fields must be set.
type OrderRequest struct {
	// AccountID is the TradeStation account ID the order is placed against.
	AccountID string `json:"AccountID"`

	// Symbol is the security identifier (e.g. "AAPL" for stock, option
	// symbols follow TradeStation's symbology format).
	Symbol string `json:"Symbol"`

	// Quantity is the number of shares or contracts. Must be positive.
	// Supports fractional values for assets that permit them.
	Quantity StringFloat64 `json:"Quantity"`

	// OrderType — see OrderType constants for price-field requirements.
	OrderType OrderType `json:"OrderType"`

	// TradeAction — see TradeAction constants for equity vs option semantics.
	TradeAction TradeAction `json:"TradeAction"`

	// LimitPrice is required for OrderTypeLimit and OrderTypeStopLimit;
	// must not be set for Market or StopMarket.
	LimitPrice StringFloat64 `json:"LimitPrice,omitempty"`

	// StopPrice is required for OrderTypeStopMarket and OrderTypeStopLimit;
	// must not be set for Market or Limit.
	StopPrice StringFloat64 `json:"StopPrice,omitempty"`

	// TimeInForce controls how long the order remains active. Duration is
	// required; ExpirationDate is required when Duration=DurationGTD.
	TimeInForce TimeInForce `json:"TimeInForce"`

	// Route is the execution venue ID. Use OrderService.GetRoutes to list
	// valid routes. Optional — server picks a default when unset.
	Route string `json:"Route,omitempty"`

	// BuyingPowerWarning controls buying-power-exceeded handling, typically
	// "Enforce" (default) or "Preconfirmed".
	BuyingPowerWarning string `json:"BuyingPowerWarning,omitempty"`

	// AdvancedOptions is an opaque string for advanced order features
	// (trailing stops, activation triggers, all-or-none, etc.). See spec.
	AdvancedOptions string `json:"AdvancedOptions,omitempty"`

	// OrderConfirmID references a prior PlaceOrderConfirm response,
	// binding placement to a previewed order for safety.
	OrderConfirmID string `json:"OrderConfirmID,omitempty"`

	// Legs populates multi-leg (option spread) orders. Must use option
	// TradeActions (BUYTOOPEN / SELLTOOPEN / BUYTOCLOSE / SELLTOCLOSE).
	Legs []OrderLegRequest `json:"Legs,omitempty"`

	// OSOs (order-sends-order) fire child order groups only if this parent
	// order fills. Nested OSOs are allowed.
	OSOs []OSOOrderRequest `json:"OSOs,omitempty"`
}

// TimeInForce controls how long an order remains active before
// automatic cancellation.
type TimeInForce struct {
	// Duration is required; see Duration constants for semantics.
	Duration Duration `json:"Duration"`
	// ExpirationDate is an ISO8601 date (e.g. "2026-12-31"). Required when
	// Duration is DurationGTD; ignored for other durations.
	ExpirationDate string `json:"ExpirationDate,omitempty"`
}

// OrderLegRequest describes one leg of a multi-leg option order.
// TradeAction must be an option-variant action.
type OrderLegRequest struct {
	Symbol         string        `json:"Symbol"`
	Quantity       StringFloat64 `json:"Quantity"`
	TradeAction    TradeAction   `json:"TradeAction"`
	AssetType      string        `json:"AssetType,omitempty"`
	ExpirationDate string        `json:"ExpirationDate,omitempty"` // option expiration
	StrikePrice    StringFloat64 `json:"StrikePrice,omitempty"`
	OptionType     string        `json:"OptionType,omitempty"` // CALL | PUT
}

// OSOOrderRequest (order-sends-order) describes a child order group that
// activates only when the parent order fills. The child group itself has a
// Type (bracket, OCO, or normal) and contains 1+ orders. Chaining is allowed
// via OrderRequest.OSOs on the child orders.
type OSOOrderRequest struct {
	Type   OrderGroupType `json:"Type"`
	Orders []OrderRequest `json:"Orders"`
}

// ReplaceOrderRequest is the body of PUT /orders/{orderID}. Contains only
// fields the spec allows to modify on an open order. All fields are optional;
// at least one must be set or the request is rejected at the validation
// boundary.
type ReplaceOrderRequest struct {
	Quantity        StringFloat64 `json:"Quantity,omitempty"`
	LimitPrice      StringFloat64 `json:"LimitPrice,omitempty"`
	StopPrice       StringFloat64 `json:"StopPrice,omitempty"`
	TimeInForce     *TimeInForce  `json:"TimeInForce,omitempty"`
	AdvancedOptions string        `json:"AdvancedOptions,omitempty"`
}

// OrderGroupRequest is the body of POST /ordergroups and POST /ordergroupsconfirm.
// Must contain at least 2 orders.
type OrderGroupRequest struct {
	// Type determines how the Orders relate after placement.
	// See OrderGroupType constants.
	Type OrderGroupType `json:"Type"`
	// Orders is the list of orders in the group (2+ required).
	Orders []OrderRequest `json:"Orders"`
}

// OrderType is the execution style for an order. Different OrderTypes require
// different price fields on OrderRequest:
//
//   - OrderTypeMarket:     no price fields
//   - OrderTypeLimit:      LimitPrice required
//   - OrderTypeStopMarket: StopPrice required
//   - OrderTypeStopLimit:  both LimitPrice and StopPrice required
type OrderType string

const (
	// OrderTypeMarket executes immediately at the best available price.
	// Must not set LimitPrice or StopPrice.
	OrderTypeMarket OrderType = "Market"

	// OrderTypeLimit executes only at LimitPrice or better.
	// Requires OrderRequest.LimitPrice > 0.
	OrderTypeLimit OrderType = "Limit"

	// OrderTypeStopMarket becomes a market order once the last trade reaches
	// StopPrice. Requires OrderRequest.StopPrice > 0.
	OrderTypeStopMarket OrderType = "StopMarket"

	// OrderTypeStopLimit becomes a limit order once StopPrice is reached,
	// then executes only at LimitPrice or better. Requires both
	// OrderRequest.StopPrice > 0 and OrderRequest.LimitPrice > 0.
	OrderTypeStopLimit OrderType = "StopLimit"
)

// TradeAction names the buy/sell side of an order. Equity and option trades
// use different action sets:
//
//   - Equities: BUY, SELL, BUYTOCOVER, SELLSHORT
//   - Options:  BUYTOOPEN, BUYTOCLOSE, SELLTOOPEN, SELLTOCLOSE
//
// Using an option action on an equity order (or vice versa) will be rejected
// by the server.
type TradeAction string

const (
	// TradeActionBuy opens or increases a long equity position.
	TradeActionBuy TradeAction = "BUY"
	// TradeActionSell closes or reduces a long equity position.
	TradeActionSell TradeAction = "SELL"
	// TradeActionBuyToCover closes a short equity position by buying back
	// the borrowed shares.
	TradeActionBuyToCover TradeAction = "BUYTOCOVER"
	// TradeActionSellShort opens or increases a short equity position.
	// Requires a margin account and locatable borrow.
	TradeActionSellShort TradeAction = "SELLSHORT"
	// TradeActionBuyToOpen opens a new long option position. Options only.
	TradeActionBuyToOpen TradeAction = "BUYTOOPEN"
	// TradeActionBuyToClose closes an existing short option position. Options only.
	TradeActionBuyToClose TradeAction = "BUYTOCLOSE"
	// TradeActionSellToOpen opens a new short option position (write). Options only.
	TradeActionSellToOpen TradeAction = "SELLTOOPEN"
	// TradeActionSellToClose closes an existing long option position. Options only.
	TradeActionSellToClose TradeAction = "SELLTOCLOSE"
)

// Duration is the time-in-force policy — how long an order remains active
// before automatic cancellation.
type Duration string

const (
	// DurationDay: active only during the current trading session.
	DurationDay Duration = "DAY"
	// DurationGTC (good-til-canceled): active across sessions until filled or
	// explicitly canceled, subject to broker maximum GTC age.
	DurationGTC Duration = "GTC"
	// DurationGTD (good-til-date): active until TimeInForce.ExpirationDate.
	// Requires TimeInForce.ExpirationDate to be a future ISO8601 date.
	DurationGTD Duration = "GTD"
	// DurationIOC (immediate-or-cancel): any portion that can't fill immediately
	// is canceled. Partial fills allowed.
	DurationIOC Duration = "IOC"
	// DurationFOK (fill-or-kill): the entire quantity must fill immediately or
	// the order is canceled. No partial fills.
	DurationFOK Duration = "FOK"
	// DurationOPG: participates in the opening auction only.
	DurationOPG Duration = "OPG"
	// DurationCLO: participates in the closing auction only.
	DurationCLO Duration = "CLO"
)

// OrderGroupType determines how orders in a PlaceOrderGroup relate to each
// other after placement.
type OrderGroupType string

const (
	// OrderGroupTypeBracket (BRK): a parent order with one or more child
	// exits (typically a profit-target limit plus a stop-loss). When one
	// child fills or cancels, the others are automatically canceled.
	OrderGroupTypeBracket OrderGroupType = "BRK"
	// OrderGroupTypeOCO (one-cancels-other): peer orders where any fill
	// cancels the remaining orders in the group. No parent.
	OrderGroupTypeOCO OrderGroupType = "OCO"
	// OrderGroupTypeNormal: orders are submitted together but operate
	// independently (not linked).
	OrderGroupTypeNormal OrderGroupType = "NORMAL"
)

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

// PlaceOrderResponse is the response body of PlaceOrder / PlaceOrderGroup.
// Orders contains successfully placed orders; Errors contains per-order
// rejections. A 200 response may carry both.
type PlaceOrderResponse struct {
	Orders []PlacedOrder `json:"Orders"`
	Errors []OrderError  `json:"Errors,omitempty"`
}

// PlacedOrder is the thin confirmation returned by the server for each
// successfully placed order. Use Brokerage.GetOrdersByID to fetch the full
// Order payload.
type PlacedOrder struct {
	OrderID string `json:"OrderID"`
	Message string `json:"Message"`
}

// OrderError is a per-order rejection inside a placement response.
type OrderError struct {
	// OrderNumber is the index into the request's Orders array (or "0" for
	// single-order placements).
	OrderNumber string `json:"OrderNumber"`
	// ErrorCode is the machine-readable error category (e.g.
	// "InsufficientBuyingPower"). Named ErrorCode in Go to avoid collision
	// with the error interface method.
	ErrorCode string `json:"Error"`
	Message   string `json:"Message"`
}

// ConfirmationResponse is the response body of PlaceOrderConfirm /
// PlaceOrderGroupConfirm — previews without execution.
type ConfirmationResponse struct {
	Confirmations []OrderConfirmation `json:"Confirmations"`
	Errors        []OrderError        `json:"Errors,omitempty"`
}

// OrderConfirmation previews an order. Pass OrderConfirmID on a subsequent
// PlaceOrder to submit the previewed order for execution.
type OrderConfirmation struct {
	OrderConfirmID           string        `json:"OrderConfirmID"`
	Route                    string        `json:"Route"`
	Duration                 string        `json:"Duration"`
	Account                  string        `json:"Account"`
	SummaryMessage           string        `json:"SummaryMessage"`
	EstimatedCommission      StringFloat64 `json:"EstimatedCommission"`
	EstimatedPrice           StringFloat64 `json:"EstimatedPrice"`
	EstimatedPriceDisplay    string        `json:"EstimatedPriceDisplay"`
	EstimatedCost            StringFloat64 `json:"EstimatedCost"`
	EstimatedCostDisplay     string        `json:"EstimatedCostDisplay"`
	DebitCreditEstimatedCost StringFloat64 `json:"DebitCreditEstimatedCost"`
	InitialMarginDisplay     string        `json:"InitialMarginDisplay"`
	ProductCurrency          string        `json:"ProductCurrency"`
	AccountCurrency          string        `json:"AccountCurrency"`
}

// ActivationTrigger describes a valid activation-trigger identifier for
// stop / trigger orders. Retrieved via GetActivationTriggers. Use the Key
// in OrderRequest.AdvancedOptions.
type ActivationTrigger struct {
	Key         string `json:"Key"` // e.g. "STT", "DTT", "SBA"
	Name        string `json:"Name"`
	Description string `json:"Description"`
}
