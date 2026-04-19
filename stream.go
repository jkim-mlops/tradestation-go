package tradestation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
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
)

// streamEnvelope captures TradeStation's control fields. Payload-specific fields
// are ignored by this decode and parsed separately by the service layer.
type streamEnvelope struct {
	StreamStatus StreamStatus `json:"StreamStatus,omitempty"`
	Error        string       `json:"Error,omitempty"`
	Message      string       `json:"Message,omitempty"`
}

// classifyStreamMessage peeks at a raw message, returning its kind and envelope.
// For data messages the envelope is zero-valued.
func classifyStreamMessage(raw []byte) (streamMessageKind, streamEnvelope, error) {
	var env streamEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return streamMessageData, env, err
	}
	switch {
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
