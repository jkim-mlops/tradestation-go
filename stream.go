package tradestation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

const streamMaxMessageSize = 1 << 20 // 1 MiB

// StreamReader reads newline-delimited JSON messages from an io.Reader.
// It correctly reassembles messages that span multiple reads.
type StreamReader struct {
	s *bufio.Scanner
}

// NewStreamReader wraps r, framing by newline. The maximum single message size
// is streamMaxMessageSize; larger messages cause Err() to return bufio.ErrTooLong.
func NewStreamReader(r io.Reader) *StreamReader {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), streamMaxMessageSize)
	s.Split(bufio.ScanLines)
	return &StreamReader{s: s}
}

// Scan advances to the next message. Returns false on EOF or error.
func (r *StreamReader) Scan() bool { return r.s.Scan() }

// Bytes returns the current message. The slice aliases an internal buffer and
// is valid only until the next call to Scan. Callers retaining the bytes
// across Scan calls must copy them.
func (r *StreamReader) Bytes() []byte { return r.s.Bytes() }

// Err returns the first non-EOF error encountered, or nil.
func (r *StreamReader) Err() error { return r.s.Err() }

type StreamStatus string

const (
	StreamStatusNone        StreamStatus = ""
	StreamStatusEndSnapshot StreamStatus = "EndSnapshot"
	StreamStatusGoAway      StreamStatus = "GoAway"
)

type streamMessageKind int

const (
	streamMessageData streamMessageKind = iota
	streamMessageStatus
	streamMessageError
	streamMessageHeartbeat
)

// streamEnvelope captures TradeStation's control fields. Payload-specific fields
// are ignored by this decode and parsed separately by the service layer.
type streamEnvelope struct {
	StreamStatus StreamStatus `json:"StreamStatus,omitempty"`
	Error        string       `json:"Error,omitempty"`
	Message      string       `json:"Message,omitempty"`
	// Heartbeat is a pointer so we can distinguish "not a heartbeat" (nil) from
	// a Heartbeat of 0. TradeStation sends {"Heartbeat":N,"Timestamp":"..."}
	// periodically on all streaming endpoints; callers never need to see them.
	Heartbeat *int64 `json:"Heartbeat,omitempty"`
}

// classifyStreamMessage peeks at a raw message, returning its kind and envelope.
// For data messages the envelope is zero-valued.
func classifyStreamMessage(raw []byte) (streamMessageKind, streamEnvelope, error) {
	var env streamEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return streamMessageData, env, err
	}
	switch {
	case env.Heartbeat != nil:
		return streamMessageHeartbeat, env, nil
	case env.StreamStatus != "":
		return streamMessageStatus, env, nil
	case env.Error != "":
		return streamMessageError, env, nil
	default:
		return streamMessageData, env, nil
	}
}

// StreamError is returned via the event channel when TradeStation emits an
// in-stream Error message. Distinct from *APIError (transport/4xx/5xx).
type StreamError struct {
	Code    string
	Message string
	RawBody []byte
}

func (e *StreamError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("tradestation: stream error %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("tradestation: stream error %s", e.Code)
}

type streamOpts struct {
	reconnect  bool
	backoffMin time.Duration
	backoffMax time.Duration
}

// StreamOption configures streaming behavior on calls like StreamQuotes.
type StreamOption func(*streamOpts)

// WithoutReconnect disables automatic reconnection. The stream channel will
// close on EOF, GoAway, or any error without retrying.
func WithoutReconnect() StreamOption {
	return func(o *streamOpts) { o.reconnect = false }
}

// WithReconnectBackoff sets the minimum and maximum reconnect delays used by
// the exponential backoff with jitter. Min resets after a clean GoAway.
func WithReconnectBackoff(min, max time.Duration) StreamOption {
	return func(o *streamOpts) { o.backoffMin, o.backoffMax = min, max }
}

func defaultStreamOpts() streamOpts {
	return streamOpts{
		reconnect:  true,
		backoffMin: 500 * time.Millisecond,
		backoffMax: 30 * time.Second,
	}
}

// sleepCtx blocks for d or until ctx is cancelled. Returns true on full sleep,
// false on cancel.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// jitter returns d scaled by a random factor in [0.75, 1.25].
func jitter(d time.Duration) time.Duration {
	return time.Duration(float64(d) * (0.75 + rand.Float64()*0.5))
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// streamEvent is the type-agnostic event the runner emits. Exactly one of
// Raw, Status, or Err is populated per event.
type streamEvent struct {
	Raw    []byte
	Status StreamStatus
	Err    error
}

// runStreamOnce opens one connection and drains it. Return semantics:
//
//	terminal != nil  → do not reconnect (currently only *StreamError)
//	goAway == true   → clean server-initiated reconnect (reset backoff)
//	both zero        → transient (EOF or network) — caller reconnects with escalating backoff
func (c *Client) runStreamOnce(
	ctx context.Context,
	openReq func(ctx context.Context) (*http.Request, error),
	out chan<- streamEvent,
) (terminal error, goAway bool) {
	req, err := openReq(ctx)
	if err != nil {
		return nil, false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// authTransport already refreshed on 401 before we got here. Any remaining
		// non-2xx is treated as transient so the reconnect loop can retry.
		return nil, false
	}
	return c.pumpResponse(ctx, resp, out)
}

// pumpResponse drains resp.Body, classifying messages and forwarding them to
// out. Caller owns resp.Body's lifecycle (open and close).
func (c *Client) pumpResponse(
	ctx context.Context,
	resp *http.Response,
	out chan<- streamEvent,
) (terminal error, goAway bool) {
	rdr := NewStreamReader(resp.Body)
	for rdr.Scan() {
		raw := rdr.Bytes()
		if len(raw) == 0 {
			continue
		}
		kind, env, err := classifyStreamMessage(raw)
		if err != nil {
			continue
		}
		switch kind {
		case streamMessageData:
			select {
			case out <- streamEvent{Raw: bytes.Clone(raw)}:
			case <-ctx.Done():
				return nil, false
			}
		case streamMessageStatus:
			select {
			case out <- streamEvent{Status: env.StreamStatus}:
			case <-ctx.Done():
				return nil, false
			}
			if env.StreamStatus == StreamStatusGoAway {
				return nil, true
			}
		case streamMessageError:
			return &StreamError{
				Code:    env.Error,
				Message: env.Message,
				RawBody: bytes.Clone(raw),
			}, false
		case streamMessageHeartbeat:
			// keep-alive from server — no caller-visible event
		}
	}
	return nil, false
}

// runStreamFromResp is a variant of runStream that pumps the given already-open
// response for iteration zero, then delegates to runStream for reconnection.
// Used by service-level stream calls so the synchronous connect doesn't
// require a second HTTP round trip on the happy path.
func (c *Client) runStreamFromResp(
	ctx context.Context,
	resp *http.Response,
	openReq func(ctx context.Context) (*http.Request, error),
	out chan<- streamEvent,
	opts streamOpts,
) {
	terminal, _ := c.pumpResponse(ctx, resp, out)
	resp.Body.Close()

	if terminal != nil {
		select {
		case out <- streamEvent{Err: terminal}:
		case <-ctx.Done():
		}
		close(out)
		return
	}
	if !opts.reconnect || ctx.Err() != nil {
		close(out)
		return
	}
	if !sleepCtx(ctx, jitter(opts.backoffMin)) {
		close(out)
		return
	}
	c.runStream(ctx, openReq, out, opts)
}

// runStream is the top-level streaming driver: it repeatedly opens HTTP
// connections (via openReq), pumps classified events into out, and honors
// reconnect/backoff config. It closes out when it exits — either on a
// terminal error (surfaced as the last streamEvent) or on ctx cancellation.
func (c *Client) runStream(
	ctx context.Context,
	openReq func(ctx context.Context) (*http.Request, error),
	out chan<- streamEvent,
	opts streamOpts,
) {
	defer close(out)

	backoff := opts.backoffMin
	for {
		terminal, goAway := c.runStreamOnce(ctx, openReq, out)

		if terminal != nil {
			select {
			case out <- streamEvent{Err: terminal}:
			case <-ctx.Done():
			}
			return
		}
		if !opts.reconnect {
			return
		}
		if ctx.Err() != nil {
			return
		}

		delay := backoff
		if goAway {
			delay = opts.backoffMin
		}
		if !sleepCtx(ctx, jitter(delay)) {
			return
		}
		if goAway {
			backoff = opts.backoffMin
		} else {
			backoff = minDuration(backoff*2, opts.backoffMax)
		}
	}
}
