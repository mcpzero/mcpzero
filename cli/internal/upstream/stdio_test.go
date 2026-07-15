//go:build !windows

package upstream

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeStdioServer is a minimal MCP-over-stdio server: it answers initialize and
// tools/list, and—crucially—emits a server-initiated roots/list request right
// after the initialized notification. The CLI must answer roots/list locally
// (it advertises no roots capability); the server acknowledges the reply by
// emitting a notification so the test can observe the loop completed.
const fakeStdioServer = `
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":0,"result":{"capabilities":{}}}'
      ;;
    *'notifications/initialized'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":"s1","method":"roots/list"}'
      ;;
    *'"id":"s1"'*)
      printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/roots_declined"}'
      ;;
    *'"method":"tools/list"'*)
      id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
      printf '%s\n' "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"tools\":[]}}"
      ;;
  esac
done
`

func TestStdioBidirectional(t *testing.T) {
	up := NewStdio(fakeStdioServer, "", nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := up.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer up.Close()

	// 1. roots/list must be answered locally (not forwarded). We observe the
	// server's follow-up notification, emitted only after it receives the
	// CLI's reply.
	select {
	case ev := <-up.Events():
		if strings.Contains(string(ev), `"method":"roots/list"`) {
			t.Fatalf("roots/list should be answered locally, not forwarded: %s", ev)
		}
		if !strings.Contains(string(ev), "notifications/roots_declined") {
			t.Fatalf("expected roots_declined ack on Events(), got: %s", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("did not observe roots/list auto-reply ack on Events()")
	}

	// 2. A normal client request is correlated to its response by id.
	got := make(chan string, 1)
	emit := func(m Message) error {
		got <- string(m)
		return nil
	}
	if err := up.Handle(ctx, []byte(`{"jsonrpc":"2.0","id":7,"method":"tools/list"}`), emit); err != nil {
		t.Fatalf("Handle tools/list: %v", err)
	}
	select {
	case resp := <-got:
		if jsonRPCID([]byte(resp)) != "7" {
			t.Fatalf("response not correlated by id: %s", resp)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no response for tools/list")
	}

	// 3. A client→server response (no method, e.g. the reply to roots/list) is
	// written to stdin and returns immediately without waiting for a reply.
	done := make(chan error, 1)
	go func() {
		done <- up.Handle(ctx,
			[]byte(`{"jsonrpc":"2.0","id":"s1","result":{"roots":[]}}`),
			func(Message) error { return nil },
		)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Handle client response: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Handle of a no-method response blocked waiting for a reply")
	}
}

// fakeStdioEchoID echoes the request id in tools/list responses so concurrent
// remapped ids can be correlated.
const fakeStdioEchoID = `
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":0,"result":{"capabilities":{}}}'
      ;;
    *'notifications/initialized'*)
      ;;
    *'"method":"tools/list"'*)
      id=$(printf '%s' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
      printf '%s\n' "{\"jsonrpc\":\"2.0\",\"id\":$id,\"result\":{\"tools\":[]}}"
      ;;
  esac
done
`

func TestStdioConcurrentDuplicateClientIDs(t *testing.T) {
	up := NewStdio(fakeStdioEchoID, "", nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := up.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer up.Close()

	const n = 8
	type result struct {
		idx  int
		body string
		err  error
	}
	ch := make(chan result, n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			var resp string
			err := up.Handle(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), func(m Message) error {
				resp = string(m)
				return nil
			})
			ch <- result{idx: idx, body: resp, err: err}
		}(i)
	}

	for i := 0; i < n; i++ {
		select {
		case r := <-ch:
			if r.err != nil {
				t.Fatalf("client %d: %v", r.idx, r.err)
			}
			if jsonRPCID([]byte(r.body)) != "1" {
				t.Fatalf("client %d: expected restored client id 1, got %s", r.idx, r.body)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out after %d/%d clients", i, n)
		}
	}
}
