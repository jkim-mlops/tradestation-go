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

func TestClassify_Data(t *testing.T) {
	kind, env, err := classifyStreamMessage([]byte(`{"Symbol":"AAPL","Last":150}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if kind != streamMessageData {
		t.Errorf("kind = %v, want data", kind)
	}
	if env.StreamStatus != "" || env.Error != "" {
		t.Errorf("env populated for data: %+v", env)
	}
}

func TestClassify_EndSnapshot(t *testing.T) {
	kind, env, err := classifyStreamMessage([]byte(`{"StreamStatus":"EndSnapshot"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if kind != streamMessageStatus {
		t.Errorf("kind = %v, want status", kind)
	}
	if env.StreamStatus != StreamStatusEndSnapshot {
		t.Errorf("StreamStatus = %q", env.StreamStatus)
	}
}

func TestClassify_GoAway(t *testing.T) {
	kind, env, _ := classifyStreamMessage([]byte(`{"StreamStatus":"GoAway"}`))
	if kind != streamMessageStatus || env.StreamStatus != StreamStatusGoAway {
		t.Errorf("kind=%v status=%q", kind, env.StreamStatus)
	}
}

func TestClassify_Error(t *testing.T) {
	kind, env, err := classifyStreamMessage([]byte(`{"Symbol":"AAPL","Error":"DualLogon","Message":"another client connected"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if kind != streamMessageError {
		t.Errorf("kind = %v, want error", kind)
	}
	if env.Error != "DualLogon" || env.Message == "" {
		t.Errorf("env wrong: %+v", env)
	}
}

func TestClassify_UnknownStatusPassesThrough(t *testing.T) {
	kind, env, _ := classifyStreamMessage([]byte(`{"StreamStatus":"WhoKnows"}`))
	if kind != streamMessageStatus || env.StreamStatus != "WhoKnows" {
		t.Errorf("kind=%v status=%q", kind, env.StreamStatus)
	}
}

func TestClassify_MalformedJSON(t *testing.T) {
	_, _, err := classifyStreamMessage([]byte(`{not json`))
	if err == nil {
		t.Error("want error for malformed JSON")
	}
}

func TestStreamError_Error(t *testing.T) {
	e := &StreamError{Code: "DualLogon", Message: "another client connected"}
	want := "tradestation: stream error DualLogon: another client connected"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	bare := &StreamError{Code: "Unknown"}
	if got := bare.Error(); got != "tradestation: stream error Unknown" {
		t.Errorf("Error() without message = %q", got)
	}
}
