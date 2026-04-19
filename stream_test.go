package tradestation

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func TestClassify_Heartbeat(t *testing.T) {
	kind, env, err := classifyStreamMessage([]byte(`{"Heartbeat":7,"Timestamp":"2026-04-19T16:52:56Z"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if kind != streamMessageHeartbeat {
		t.Errorf("kind = %v, want heartbeat", kind)
	}
	if env.Heartbeat == nil || *env.Heartbeat != 7 {
		t.Errorf("Heartbeat = %v, want 7", env.Heartbeat)
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

func TestDefaultStreamOpts(t *testing.T) {
	o := defaultStreamOpts()
	if !o.reconnect {
		t.Error("reconnect should default to true")
	}
	if o.backoffMin != 500*time.Millisecond {
		t.Errorf("backoffMin = %v, want 500ms", o.backoffMin)
	}
	if o.backoffMax != 30*time.Second {
		t.Errorf("backoffMax = %v, want 30s", o.backoffMax)
	}
}

func TestWithoutReconnect(t *testing.T) {
	o := defaultStreamOpts()
	WithoutReconnect()(&o)
	if o.reconnect {
		t.Error("reconnect should be false after WithoutReconnect")
	}
}

func TestWithReconnectBackoff(t *testing.T) {
	o := defaultStreamOpts()
	WithReconnectBackoff(1*time.Second, 10*time.Second)(&o)
	if o.backoffMin != 1*time.Second || o.backoffMax != 10*time.Second {
		t.Errorf("backoff wrong: min=%v max=%v", o.backoffMin, o.backoffMax)
	}
}

func TestSleepCtx_CompletesNormally(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	if !sleepCtx(ctx, 20*time.Millisecond) {
		t.Error("want true on normal completion")
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Errorf("returned too fast: %v", elapsed)
	}
}

func TestSleepCtx_CancelReturnsFalse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if sleepCtx(ctx, 1*time.Hour) {
		t.Error("want false on cancel")
	}
}

func TestJitter_StaysInRange(t *testing.T) {
	base := 100 * time.Millisecond
	for i := 0; i < 200; i++ {
		j := jitter(base)
		if j < time.Duration(float64(base)*0.75) || j > time.Duration(float64(base)*1.25) {
			t.Fatalf("jitter %v out of ±25%% range of %v", j, base)
		}
	}
}

func TestMinDuration(t *testing.T) {
	if minDuration(1*time.Second, 2*time.Second) != 1*time.Second {
		t.Error("minDuration wrong for a<b")
	}
	if minDuration(2*time.Second, 1*time.Second) != 1*time.Second {
		t.Error("minDuration wrong for a>b")
	}
}

// drainEvents reads everything from ch into a slice until it closes.
func drainEvents(ch <-chan streamEvent) []streamEvent {
	var out []streamEvent
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

// chunkedWrite writes and flushes each string as a separate HTTP chunk so
// tests can simulate messages split across TCP boundaries.
func chunkedWrite(w http.ResponseWriter, chunks ...string) {
	f, ok := w.(http.Flusher)
	if !ok {
		panic("ResponseWriter does not support flushing")
	}
	for _, c := range chunks {
		w.Write([]byte(c))
		f.Flush()
	}
}

func TestRunStreamOnce_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunkedWrite(w,
			`{"Symbol":"AAPL","Last":1}`+"\n",
			`{"Symbol":"MSFT","Last":2}`+"\n",
		)
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	ch := make(chan streamEvent, 8)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	go func() {
		defer close(ch)
		_, _ = c.runStreamOnce(context.Background(), openReq, ch)
	}()

	got := drainEvents(ch)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	for i, ev := range got {
		if ev.Err != nil || ev.Status != "" {
			t.Errorf("ev[%d] not data: %+v", i, ev)
		}
	}
}

func TestRunStreamOnce_GoAwayReturnsTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunkedWrite(w,
			`{"Symbol":"AAPL"}`+"\n",
			`{"StreamStatus":"GoAway"}`+"\n",
			`{"Symbol":"SHOULD-NOT-APPEAR"}`+"\n",
		)
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	ch := make(chan streamEvent, 8)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}

	terminalCh := make(chan error, 1)
	goAwayCh := make(chan bool, 1)
	go func() {
		defer close(ch)
		term, ga := c.runStreamOnce(context.Background(), openReq, ch)
		terminalCh <- term
		goAwayCh <- ga
	}()

	got := drainEvents(ch)
	if term := <-terminalCh; term != nil {
		t.Errorf("terminal = %v, want nil", term)
	}
	if !<-goAwayCh {
		t.Error("goAway should be true")
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 (data + GoAway)", len(got))
	}
	if got[1].Status != StreamStatusGoAway {
		t.Errorf("last event Status = %q", got[1].Status)
	}
}

func TestRunStreamOnce_ErrorIsTerminal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunkedWrite(w,
			`{"Symbol":"AAPL"}`+"\n",
			`{"Error":"DualLogon","Message":"another client"}`+"\n",
		)
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	ch := make(chan streamEvent, 8)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	terminalCh := make(chan error, 1)
	go func() {
		defer close(ch)
		term, _ := c.runStreamOnce(context.Background(), openReq, ch)
		terminalCh <- term
	}()

	got := drainEvents(ch)
	term := <-terminalCh
	var se *StreamError
	if !errors.As(term, &se) {
		t.Fatalf("terminal not *StreamError: %v", term)
	}
	if se.Code != "DualLogon" {
		t.Errorf("Code = %q", se.Code)
	}
	if len(got) != 1 {
		t.Errorf("got %d events, want 1 (only the pre-error data)", len(got))
	}
}

func TestRunStreamOnce_NonSuccessStatusIsTransient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	ch := make(chan streamEvent, 2)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}

	term, ga := c.runStreamOnce(context.Background(), openReq, ch)
	close(ch)
	if term != nil {
		t.Errorf("terminal = %v, want nil (transient)", term)
	}
	if ga {
		t.Error("goAway should be false")
	}
}

func TestRunStream_ReconnectsAfterEOF(t *testing.T) {
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&calls, 1)
		chunkedWrite(w, `{"call":`+string(rune('0'+n))+`}`+"\n")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	out := make(chan streamEvent)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.backoffMin = 1 * time.Millisecond
	opts.backoffMax = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.runStream(ctx, openReq, out, opts)

	got := make([]streamEvent, 0, 3)
	for ev := range out {
		got = append(got, ev)
		if len(got) == 3 {
			cancel()
		}
	}

	if atomic.LoadInt64(&calls) < 3 {
		t.Errorf("calls = %d, want >= 3 (reconnected)", calls)
	}
	if len(got) < 3 {
		t.Errorf("got %d events, want >= 3", len(got))
	}
}

func TestRunStream_GoAwayReconnects(t *testing.T) {
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		chunkedWrite(w, `{"StreamStatus":"GoAway"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	out := make(chan streamEvent)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.backoffMin = 1 * time.Millisecond
	opts.backoffMax = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.runStream(ctx, openReq, out, opts)

	var statuses int
	for ev := range out {
		if ev.Status == StreamStatusGoAway {
			statuses++
			if statuses == 3 {
				cancel()
			}
		}
	}
	if statuses < 3 {
		t.Errorf("statuses = %d, want >= 3", statuses)
	}
}

func TestRunStream_ErrorIsTerminalAndClosesChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunkedWrite(w, `{"Error":"DualLogon","Message":"boom"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	out := make(chan streamEvent)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.backoffMin = 1 * time.Millisecond

	go c.runStream(context.Background(), openReq, out, opts)

	got := drainEvents(out)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1 (terminal only)", len(got))
	}
	var se *StreamError
	if !errors.As(got[0].Err, &se) {
		t.Fatalf("not a *StreamError: %v", got[0].Err)
	}
	if se.Code != "DualLogon" {
		t.Errorf("Code = %q", se.Code)
	}
}

func TestRunStream_WithoutReconnectStopsOnEOF(t *testing.T) {
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		chunkedWrite(w, `{"Symbol":"AAPL"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	out := make(chan streamEvent)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.reconnect = false

	go c.runStream(context.Background(), openReq, out, opts)

	got := drainEvents(out)
	if len(got) != 1 {
		t.Errorf("got %d events, want 1", len(got))
	}
	if atomic.LoadInt64(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (no reconnect)", calls)
	}
}

func TestRunStream_ContextCancelClosesChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	out := make(chan streamEvent)
	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.backoffMin = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		c.runStream(ctx, openReq, out, opts)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runStream did not exit after cancel")
	}
}

func TestRunStreamFromResp_UsesPreOpenedResponse(t *testing.T) {
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		chunkedWrite(w, `{"Symbol":"AAPL"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.reconnect = false

	out := make(chan streamEvent, 4)
	go c.runStreamFromResp(context.Background(), resp, openReq, out, opts)

	got := drainEvents(out)
	if len(got) != 1 {
		t.Errorf("got %d events, want 1", len(got))
	}
	if atomic.LoadInt64(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (no re-open)", calls)
	}
}

func TestRunStreamFromResp_ReconnectsOnEOF(t *testing.T) {
	var calls int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		chunkedWrite(w, `{"Symbol":"AAPL"}`+"\n")
	}))
	defer srv.Close()

	c := NewClient(Test, "id", "secret", "refresh")
	c.apiBase = srv.URL

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	openReq := func(ctx context.Context) (*http.Request, error) {
		return http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	}
	opts := defaultStreamOpts()
	opts.backoffMin = 1 * time.Millisecond
	opts.backoffMax = 5 * time.Millisecond

	out := make(chan streamEvent)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.runStreamFromResp(ctx, resp, openReq, out, opts)

	var seen int
	for range out {
		seen++
		if seen == 3 {
			cancel()
		}
	}
	if atomic.LoadInt64(&calls) < 3 {
		t.Errorf("calls = %d, want >= 3 (re-opened after EOF)", calls)
	}
}
