package tradestation

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestStreamReader_SingleMessage(t *testing.T) {
	r := NewStreamReader(strings.NewReader(`{"a":1}` + "\n"))
	if !r.Scan() {
		t.Fatalf("Scan returned false, Err=%v", r.Err())
	}
	if got := string(r.Bytes()); got != `{"a":1}` {
		t.Errorf("Bytes = %q", got)
	}
	if r.Scan() {
		t.Error("expected EOF")
	}
}

func TestStreamReader_MultipleMessagesInOneRead(t *testing.T) {
	r := NewStreamReader(strings.NewReader(`{"a":1}` + "\n" + `{"a":2}` + "\n" + `{"a":3}` + "\n"))
	var got []string
	for r.Scan() {
		got = append(got, string(r.Bytes()))
	}
	if err := r.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
	want := []string{`{"a":1}`, `{"a":2}`, `{"a":3}`}
	if len(got) != len(want) {
		t.Fatalf("got %d msgs, want %d: %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("msg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStreamReader_MessageSplitAcrossReads(t *testing.T) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		pw.Write([]byte(`{"hel`))
		pw.Write([]byte(`lo":"world"}` + "\n"))
	}()
	r := NewStreamReader(pr)
	if !r.Scan() {
		t.Fatalf("Scan returned false, Err=%v", r.Err())
	}
	if got := string(r.Bytes()); got != `{"hello":"world"}` {
		t.Errorf("Bytes = %q", got)
	}
}

func TestStreamReader_TrailingMessageWithoutNewline(t *testing.T) {
	r := NewStreamReader(strings.NewReader(`{"a":1}` + "\n" + `{"a":2}`))
	var got []string
	for r.Scan() {
		got = append(got, string(r.Bytes()))
	}
	want := []string{`{"a":1}`, `{"a":2}`}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStreamReader_EmptyLinesSkipped(t *testing.T) {
	r := NewStreamReader(strings.NewReader("\n\n" + `{"a":1}` + "\n\n" + `{"a":2}` + "\n"))
	var got []string
	for r.Scan() {
		if b := r.Bytes(); len(b) > 0 {
			got = append(got, string(b))
		}
	}
	if len(got) != 2 {
		t.Errorf("got %d non-empty msgs, want 2: %v", len(got), got)
	}
}

func TestStreamReader_MaxSizeExceeded(t *testing.T) {
	big := strings.Repeat("x", streamMaxMessageSize+1)
	r := NewStreamReader(strings.NewReader(big + "\n"))
	if r.Scan() {
		t.Fatal("expected Scan to fail on oversized input")
	}
	if err := r.Err(); !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("Err = %v, want bufio.ErrTooLong", err)
	}
}
