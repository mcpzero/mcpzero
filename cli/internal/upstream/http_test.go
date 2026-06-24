package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func collect(t *testing.T, up Upstream, body string) []string {
	t.Helper()
	var got []string
	err := up.Handle(context.Background(), []byte(body), func(m Message) error {
		got = append(got, string(m))
		return nil
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	return got
}

func TestHTTPUpstreamHandleJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
	}))
	defer srv.Close()

	up, err := NewHTTP(srv.URL, nil, TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, up, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if len(got) != 1 || got[0] != `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}` {
		t.Fatalf("unexpected messages: %#v", got)
	}
}

func TestHTTPUpstreamHandleSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("event: message\ndata: {\"id\":1,\"result\":{\"progress\":1}}\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("event: message\ndata: {\"id\":1,\"result\":{\"done\":true}}\n\n"))
		flusher.Flush()
	}))
	defer srv.Close()

	up, err := NewHTTP(srv.URL, nil, TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, up, `{"jsonrpc":"2.0","id":1,"method":"tools/call"}`)
	if len(got) != 2 {
		t.Fatalf("expected 2 streamed messages, got %d: %#v", len(got), got)
	}
	if got[1] != `{"id":1,"result":{"done":true}}` {
		t.Fatalf("unexpected final message: %s", got[1])
	}
}

func TestHTTPUpstreamSessionAndHeaders(t *testing.T) {
	var sawAuth, sawSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body [4096]byte
		n, _ := r.Body.Read(body[:])
		isInit := strings.Contains(string(body[:n]), `"method":"initialize"`)
		if isInit {
			w.Header().Set("Mcp-Session-Id", "sess-123")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{}}`))
			return
		}
		sawAuth = r.Header.Get("Authorization")
		sawSession = r.Header.Get("Mcp-Session-Id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
	}))
	defer srv.Close()

	up, err := NewHTTP(srv.URL, []Header{{Name: "Authorization", Value: "Bearer t"}}, TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}
	// Drive a single Handle that should carry no session yet, then initialize
	// captures the session for the following call.
	if err := up.Handle(context.Background(), []byte(initializeRequest), func(Message) error { return nil }); err != nil {
		t.Fatalf("initialize handle: %v", err)
	}
	collect(t, up, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	if sawAuth != "Bearer t" {
		t.Fatalf("auth header not forwarded: %q", sawAuth)
	}
	if sawSession != "sess-123" {
		t.Fatalf("session id not replayed: %q", sawSession)
	}
}

func TestHTTPUpstreamNotificationNoBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	up, err := NewHTTP(srv.URL, nil, TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}
	got := collect(t, up, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if len(got) != 0 {
		t.Fatalf("expected no messages for notification, got %#v", got)
	}
}

func TestParseSSEStreamMultiline(t *testing.T) {
	input := "event: endpoint\ndata: /messages?id=1\n\n: comment\ndata: line1\ndata: line2\n\n"
	var events [][2]string
	err := parseSSEStream(context.Background(), strings.NewReader(input), func(ev, data string) error {
		events = append(events, [2]string{ev, data})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %#v", len(events), events)
	}
	if events[0] != [2]string{"endpoint", "/messages?id=1"} {
		t.Fatalf("unexpected endpoint event: %#v", events[0])
	}
	if events[1] != [2]string{"message", "line1\nline2"} {
		t.Fatalf("unexpected multiline event: %#v", events[1])
	}
}

func TestHandleContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		time.Sleep(2 * time.Second)
	}))
	defer srv.Close()

	up, err := NewHTTP(srv.URL, nil, TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = up.Handle(ctx, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call"}`), func(Message) error { return nil })
	// Should return promptly once ctx is canceled rather than hang for 2s.
}
