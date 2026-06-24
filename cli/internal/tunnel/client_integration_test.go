package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mcpzero/mcpzero/cli/internal/upstream"
)

// TestTunnelHTTPUpstreamStreaming drives the real Client against a mock gateway
// (WebSocket) and a mock HTTP MCP server, asserting that a forwarded request is
// proxied to the upstream and streamed back as chunk + end over protocol v2.
func TestTunnelHTTPUpstreamStreaming(t *testing.T) {
	// Mock HTTP MCP server: JSON for initialize, SSE for tools/call.
	mcp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed) // no server event stream
			return
		}
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		if strings.Contains(string(body), `"method":"initialize"`) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":7,"result":{"sum":3}}`))
	}))
	defer mcp.Close()

	type result struct {
		registered bool
		transport  string
		chunks     []string
		ended      bool
	}
	done := make(chan result, 1)

	upgrader := websocket.Upgrader{}
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var res result

		// Read register.
		_, data, err := conn.ReadMessage()
		if err != nil {
			done <- res
			return
		}
		var reg map[string]any
		_ = json.Unmarshal(data, &reg)
		if reg["type"] == "register" {
			res.registered = true
			if tr, ok := reg["transport"].(string); ok {
				res.transport = tr
			}
		}
		_ = conn.WriteJSON(map[string]any{"type": "register_ok"})

		// Forward one request.
		_ = conn.WriteJSON(map[string]string{
			"type": "mcp_request",
			"id":   "r1",
			"body": `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"add","arguments":{"a":1,"b":2}}}`,
		})

		// Collect until stream end.
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg["type"] {
			case "mcp_stream_chunk":
				if msg["id"] == "r1" {
					res.chunks = append(res.chunks, msg["body"].(string))
				}
			case "mcp_stream_end":
				if msg["id"] == "r1" {
					res.ended = true
					done <- res
					return
				}
			}
		}
		done <- res
	}))
	defer gw.Close()

	up, err := upstream.NewHTTP(mcp.URL, nil, upstream.TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}

	client := Client{
		GWBase:     gw.URL,
		EndpointID: "ep_test",
		Token:      "t",
		Upstream:   up,
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- client.Run(ctx) }()

	select {
	case res := <-done:
		if !res.registered {
			t.Fatalf("gateway did not receive register")
		}
		if res.transport != "streamable-http" {
			t.Fatalf("expected transport streamable-http, got %q", res.transport)
		}
		if !res.ended {
			t.Fatalf("stream did not end")
		}
		if len(res.chunks) != 1 || res.chunks[0] != `{"jsonrpc":"2.0","id":7,"result":{"sum":3}}` {
			t.Fatalf("unexpected chunks: %#v", res.chunks)
		}
	case <-time.After(8 * time.Second):
		cancel()
		t.Fatal("timed out waiting for tunnel to stream response")
	}

	cancel()
	select {
	case <-runErr:
	case <-time.After(3 * time.Second):
		t.Fatal("client.Run did not return after cancel")
	}
}

// TestTunnelMultiServerRouting drives a Client multiplexing two named HTTP
// upstreams and asserts that register advertises both server names and that an
// mcp_request tagged with a server name is routed to the matching upstream.
func TestTunnelMultiServerRouting(t *testing.T) {
	newMCP := func(tag string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			body := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(body)
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), `"method":"initialize"`) {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{}}`))
				return
			}
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":7,"result":{"who":"` + tag + `"}}`))
		}))
	}
	alpha := newMCP("alpha")
	defer alpha.Close()
	beta := newMCP("beta")
	defer beta.Close()

	type result struct {
		servers []string
		chunk   string
		ended   bool
	}
	done := make(chan result, 1)

	upgrader := websocket.Upgrader{}
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var res result
		_, data, err := conn.ReadMessage()
		if err != nil {
			done <- res
			return
		}
		var reg struct {
			Type    string   `json:"type"`
			Servers []string `json:"servers"`
		}
		_ = json.Unmarshal(data, &reg)
		res.servers = reg.Servers
		_ = conn.WriteJSON(map[string]any{"type": "register_ok"})

		// Forward a request explicitly targeting the "beta" server.
		_ = conn.WriteJSON(map[string]string{
			"type":   "mcp_request",
			"id":     "r1",
			"server": "beta",
			"body":   `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"who","arguments":{}}}`,
		})

		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var msg map[string]any
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			switch msg["type"] {
			case "mcp_stream_chunk":
				if msg["id"] == "r1" {
					res.chunk = msg["body"].(string)
				}
			case "mcp_stream_end":
				if msg["id"] == "r1" {
					res.ended = true
					done <- res
					return
				}
			}
		}
		done <- res
	}))
	defer gw.Close()

	upAlpha, err := upstream.NewHTTP(alpha.URL, nil, upstream.TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}
	upBeta, err := upstream.NewHTTP(beta.URL, nil, upstream.TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}

	client := Client{
		GWBase:     gw.URL,
		EndpointID: "ep_test",
		Token:      "t",
		Upstreams: []NamedUpstream{
			{Name: "alpha", Upstream: upAlpha},
			{Name: "beta", Upstream: upBeta},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- client.Run(ctx) }()

	select {
	case res := <-done:
		if len(res.servers) != 2 || res.servers[0] != "alpha" || res.servers[1] != "beta" {
			t.Fatalf("register did not advertise both servers: %#v", res.servers)
		}
		if !res.ended {
			t.Fatalf("stream did not end")
		}
		if !strings.Contains(res.chunk, `"who":"beta"`) {
			t.Fatalf("request not routed to beta upstream: %q", res.chunk)
		}
	case <-time.After(8 * time.Second):
		cancel()
		t.Fatal("timed out waiting for multi-server response")
	}

	cancel()
	select {
	case <-runErr:
	case <-time.After(3 * time.Second):
		t.Fatal("client.Run did not return after cancel")
	}
}

// TestTunnelKickedDoesNotReconnect asserts that when the gateway closes the
// tunnel with a normal-closure "replaced_by_new_connection" frame (the eviction
// of an older tunnel), the client stops terminally and never dials again.
func TestTunnelKickedDoesNotReconnect(t *testing.T) {
	// Minimal MCP upstream that just answers initialize.
	mcp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{}}`))
	}))
	defer mcp.Close()

	var mu sync.Mutex
	conns := 0

	upgrader := websocket.Upgrader{}
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		conns++
		mu.Unlock()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Accept the register, then immediately kick like a replacement would.
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{"type": "register_ok"})
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "replaced_by_new_connection"),
			time.Now().Add(time.Second),
		)
		// Hold the socket briefly so the client reads the close frame.
		time.Sleep(150 * time.Millisecond)
	}))
	defer gw.Close()

	up, err := upstream.NewHTTP(mcp.URL, nil, upstream.TransportStreamable)
	if err != nil {
		t.Fatal(err)
	}

	client := Client{GWBase: gw.URL, EndpointID: "ep_test", Token: "t", Upstream: up}

	runErr := make(chan error, 1)
	go func() { runErr <- client.Run(context.Background()) }()

	select {
	case err := <-runErr:
		var de *DisconnectError
		if !errors.As(err, &de) || !de.Terminal {
			t.Fatalf("expected terminal disconnect after kick, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client did not stop after being kicked")
	}

	// The first reconnect would wait ~1s; give it room to (wrongly) fire.
	time.Sleep(1500 * time.Millisecond)
	mu.Lock()
	got := conns
	mu.Unlock()
	if got != 1 {
		t.Fatalf("expected exactly 1 gateway connection (no reconnect), got %d", got)
	}
}
