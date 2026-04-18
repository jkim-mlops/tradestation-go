# tradestation-go

Go client for the [TradeStation API v3](https://api.tradestation.com/docs/specification).

Stdlib-only. Automatic access-token refresh on 401. Rotating-refresh-token support. Sandbox (`sim-api.tradestation.com`) and production (`api.tradestation.com`) environments from a single constructor.

## Status

| Area | Status |
|---|---|
| OAuth / token refresh | implemented (rotate callback supported) |
| Market Data — REST (`GetBars`, `GetQuote`) | implemented |
| Market Data — Streaming, Options chain | stubbed (`panic("not implemented")`) |
| Brokerage — all 8 REST endpoints | implemented |
| Brokerage — Streaming | not started |
| Order Execution (place / replace / cancel) | stubbed |
| CLI tools (`authorize`, `envgen`) | implemented |

## Install

```
go get github.com/jkim-mlops/tradestation-go
```

Requires Go 1.25+.

## Authentication

TradeStation uses OAuth 2.0 with rotating refresh tokens. You need:

1. A client ID + client secret (from your [TradeStation developer account](https://api.tradestation.com/docs/fundamentals/authentication/auth-overview))
2. A refresh token — obtained once via the interactive authorize flow, then persisted.

The library never touches the authorize step. It only holds the refresh token and exchanges it for access tokens (automatically, on 401). If the server returns a new refresh token during that exchange, your `WithRefreshTokenRotate` callback fires so you can persist it.

### Getting a refresh token (one-time)

This repo ships a small CLI, `cmd/authorize`, that runs a local callback server, opens the browser to TradeStation's consent screen, completes the OAuth code exchange, and writes the resulting refresh token to AWS SSM Parameter Store:

```
go run ./cmd/authorize \
  -id     /tradestation/client-id \
  -secret /tradestation/client-secret \
  -refresh /tradestation/refresh-token \
  -scopes "openid offline_access MarketData ReadAccount Trade profile"
```

Reads client-id/secret from SSM, writes the refresh token back as a `SecureString`. Uses `AWS_PROFILE` if set, otherwise `joe-prod`.

### Local `.env` for integration tests

`cmd/envgen` pulls the three SSM parameters into a `.env` file used by `go test -tags=integration`:

```
go run ./cmd/envgen -out .env
```

Produces:

```
TRADESTATION_CLIENT_ID=...
TRADESTATION_CLIENT_SECRET=...
TRADESTATION_REFRESH_TOKEN=...
```

## Quickstart

```go
package main

import (
    "context"
    "fmt"

    "github.com/jkim-mlops/tradestation-go"
)

func main() {
    c := tradestation.NewClient(
        tradestation.Test, // or tradestation.Production
        "client-id",
        "client-secret",
        "refresh-token",
    )

    // Get bars for AAPL (last 5 daily).
    bars, err := c.MarketData().GetBars(context.Background(), "AAPL", tradestation.GetBarsParams{
        Interval: 1,
        Unit:     tradestation.BarUnitDaily,
        BarsBack: 5,
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("got %d bars\n", len(bars))
}
```

Services hang off `Client`: `c.MarketData()` returns a `*MarketDataService`, `c.Brokerage()` returns a `*BrokerageService`. Accessors are cheap — call them per-use or stash the result, whichever reads better.

## Environments

```go
tradestation.Test       // https://sim-api.tradestation.com — paper trading sandbox
tradestation.Production // https://api.tradestation.com
```

Both environments authenticate against the same `signin.tradestation.com` OAuth endpoint with the same credentials.

## Client options

```go
c := tradestation.NewClient(env, id, secret, refresh,
    tradestation.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
    tradestation.WithRefreshTokenRotate(func(newToken string) {
        // Persist the rotated refresh token (e.g., back to SSM).
    }),
)
```

- **`WithHTTPClient`** — supply a custom `*http.Client`. Its `Transport` is wrapped with the library's auth transport so 401 → refresh → retry still applies.
- **`WithRefreshTokenRotate`** — callback invoked when TradeStation issues a new refresh token (rotating-refresh-token flows). Called with the mutex released; safe to do I/O.

## Brokerage API

`BrokerageService` implements the eight REST endpoints under `/v3/brokerage`. All methods take `context.Context` as the first argument.

### Endpoints

| Method | HTTP | Path |
|---|---|---|
| `GetAccounts(ctx)` | GET | `/v3/brokerage/accounts` |
| `GetBalances(ctx, accountIDs)` | GET | `/v3/brokerage/accounts/{ids}/balances` |
| `GetBalancesBOD(ctx, accountIDs)` | GET | `/v3/brokerage/accounts/{ids}/bodbalances` |
| `GetPositions(ctx, accountIDs, opts...)` | GET | `/v3/brokerage/accounts/{ids}/positions` |
| `GetOrders(ctx, accountIDs)` | GET | `/v3/brokerage/accounts/{ids}/orders` |
| `GetOrdersByID(ctx, accountIDs, orderIDs)` | GET | `/v3/brokerage/accounts/{ids}/orders/{oids}` |
| `GetHistoricalOrders(ctx, accountIDs, since, opts...)` | GET | `/v3/brokerage/accounts/{ids}/historicalorders` |
| `GetHistoricalOrdersByID(ctx, accountIDs, orderIDs, since, opts...)` | GET | `/v3/brokerage/accounts/{ids}/historicalorders/{oids}` |

### Batch-by-default

Every account-scoped endpoint takes `[]string` for account IDs and joins them with commas into the URL, matching the TradeStation spec. Maximum 25 IDs per request; the client rejects more before sending.

```go
resp, err := svc.GetBalances(ctx, []string{"11111111", "22222222"})
```

### Partial per-account errors

The TradeStation API returns `200 OK` even when some requested accounts fail (closed, unauthorized, etc.). Those failures are returned inline in the response's `Errors` field — not via a Go `error`.

```go
resp, err := svc.GetBalances(ctx, accountIDs)
if err != nil {
    // transport / 4xx / 5xx / decode error
}
for _, b := range resp.Balances {
    // successful accounts
}
for _, e := range resp.Errors {
    // partial failures: e.AccountID, e.ErrorCode, e.Message
}
```

This applies to `GetBalances`, `GetBalancesBOD`, `GetPositions`, `GetOrders`, `GetOrdersByID`, `GetHistoricalOrders[ByID]`.

### Pagination (historical orders)

`GetHistoricalOrders` and `GetHistoricalOrdersByID` auto-paginate using the `NextToken` cursor the API returns. Pages are concatenated into a single `OrdersResponse` (orders + errors both aggregated).

```go
since := time.Now().AddDate(0, 0, -90)
resp, err := svc.GetHistoricalOrders(ctx, accountIDs, since,
    tradestation.WithMaxPages(10),    // cap — required for unbounded queries
    tradestation.WithPageSize(200),   // optional page size hint
)
```

- **`WithMaxPages(n)`** — stop after `n` pages even if the server still returns a `NextToken`. Strongly recommended; TradeStation has returned long cursor chains in practice.
- **`WithPageSize(n)`** — sets the `pageSize` query parameter per page.

`since` is formatted as `YYYY-MM-DD` in UTC.

### Positions — symbol filter

```go
resp, err := svc.GetPositions(ctx, accountIDs, tradestation.WithSymbol("AAPL*"))
```

Wildcards are passed through to the API unchanged (URL-encoded in transit).

## Market Data API

```go
// Daily bars.
bars, err := md.GetBars(ctx, "AAPL", tradestation.GetBarsParams{
    Interval: 1,
    Unit:     tradestation.BarUnitDaily,
    BarsBack: 30,
})

// Batch quote (1–50 symbols).
quotes, err := md.GetQuote(ctx, []string{"AAPL", "MSFT", "SPY"})
```

`BarUnit` values: `BarUnitMinute`, `BarUnitDaily`, `BarUnitWeekly`, `BarUnitMonthly`.

`GetBarsParams.StartDate` accepts an ISO-8601 string as an alternative to `BarsBack`.

## Numeric decoding

The TradeStation API returns many numeric fields as JSON strings (e.g. `"1000.50"`). The library's `StringFloat64` / `StringInt64` wrapper types transparently accept both string and number forms — you read them as plain `float64` / `int64`:

```go
var b Balance
b.CashBalance // StringFloat64 — compare directly: b.CashBalance > 1000
```

This applies to balance, position, order, bar, and quote fields throughout the package.

## Testing

### Unit tests (no credentials needed)

```
go test ./...
```

Brokerage tests use `net/http/httptest` fakes; the `apiBase` is swappable per-test. Runs in under a second.

### Integration tests (hits the sandbox)

```
go test -tags=integration -run TestIntegration_ -v ./...
```

- Requires `TRADESTATION_CLIENT_ID`, `TRADESTATION_CLIENT_SECRET`, `TRADESTATION_REFRESH_TOKEN`.
- Auto-loads `.env` from the current directory if present (use `cmd/envgen` to generate).
- Skips (not fails) when credentials are missing.
- Integration tests log the full JSON response so you can manually inspect the decoded sandbox data.

### Other commands

```
go vet ./...                 # static checks
go build ./...               # full build
go test -run TestName -v     # single test
```

## Project conventions

- **`docs/` is gitignored** — design docs and plans are local-only.
- **No operational dependencies in the root package** — AWS SDK, env loaders, etc. live under `cmd/<tool>/`, never in `tradestation.*`.
- **Commit style:** Conventional Commits + [gitmoji](https://gitmoji.dev/), lowercase: `type(scope): :emoji: short description`.
- **Branching:** feature work happens on `feat/<name>` branches; never commit directly to `main`.

## License

See repo / go.mod.
