//go:build !windows

package upstream

import (
	"context"
	"testing"
	"time"
)

// fakeStdioServer is a minimal MCP-over-stdio server: it answers initialize and
// tools/list, and—crucially—emits a server-initiated roots/list request right
// after the initialized notification, which is the bidirectional case the
// tunnel must now handle.
const fakeStdioServer = `
while IFS= read -r line; do
  case "$line" in
    *'"method":"initialize"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":0,"result":{"capabilities":{}}}'
      ;;
    *'notifications/initialized'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":"s1","method":"roots/list"}'
      ;;
    *'"method":"tools/list"'*)
      printf '%s\n' '{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}'
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

	// 1. The server-initiated roots/list request must surface on Events.
	select {
	case ev := <-up.Events():
		if !hasMethod(ev) || jsonRPCID(ev) != `"s1"` {
			t.Fatalf("unexpected server event: %s", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive server-initiated roots/list on Events()")
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
