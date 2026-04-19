package tradestation

import (
	"bufio"
	"io"
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
