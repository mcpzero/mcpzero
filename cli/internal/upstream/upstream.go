// Package upstream abstracts the local MCP server that a tunnel proxies. It
// supports stdio servers (launched as a subprocess) and HTTP servers (local or
// external-via-local, with local auth headers), including streaming responses.
package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/version"
)

// Message is a single JSON-RPC message (no trailing newline).
type Message = []byte

// Emit delivers one response message for an in-flight request.
type Emit func(Message) error

// Upstream is a local MCP server reachable by the tunnel.
type Upstream interface {
	// Initialize performs any handshake required before serving requests.
	Initialize(ctx context.Context) error
	// Handle processes one request body, calling emit for each response
	// message. For a notification (no id) it emits nothing and returns nil.
	Handle(ctx context.Context, reqBody []byte, emit Emit) error
	// Events returns server-initiated messages, or nil if unsupported.
	Events() <-chan Message
	// Transport reports the concrete wire transport: "stdio" for a stdio
	// subprocess, or "streamable-http" | "sse" for an HTTP upstream.
	Transport() string
	// Close releases resources (and, for stdio, reaps the subprocess).
	Close() error
}

// Header is a single HTTP header sent to an HTTP upstream.
type Header struct {
	Name  string
	Value string
}

// LoopHeader carries the gateway forwarding chain (comma-separated endpoint
// ids) so the gateway can detect request loops. The CLI propagates the value it
// received on an mcp_request to any HTTP upstream it calls, letting the gateway
// see its own endpoint reappear in the chain.
const LoopHeader = "X-MCP-Loop"

type loopCtxKey struct{}

// WithLoopTrace returns a context carrying the gateway forwarding chain for the
// current request. An empty trace leaves the context unchanged.
func WithLoopTrace(ctx context.Context, trace string) context.Context {
	if trace == "" {
		return ctx
	}
	return context.WithValue(ctx, loopCtxKey{}, trace)
}

// loopTraceFromContext returns the forwarding chain stored by WithLoopTrace, or
// "" if none is set.
func loopTraceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(loopCtxKey{}).(string); ok {
		return v
	}
	return ""
}

var envRef = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ParseHeader parses a "Name: Value" header flag and resolves ${ENV}
// references in the value from the current environment.
func ParseHeader(raw string) (Header, error) {
	idx := strings.Index(raw, ":")
	if idx <= 0 {
		return Header{}, fmt.Errorf("invalid header %q (expected \"Name: Value\")", raw)
	}
	name := strings.TrimSpace(raw[:idx])
	value := strings.TrimSpace(raw[idx+1:])
	if name == "" {
		return Header{}, fmt.Errorf("invalid header %q (empty name)", raw)
	}
	return Header{Name: name, Value: interpolateEnv(value)}, nil
}

func interpolateEnv(value string) string {
	return envRef.ReplaceAllStringFunc(value, func(match string) string {
		name := match[2 : len(match)-1]
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return match
	})
}

// expectsResponse reports whether a JSON-RPC message is a request (has both a
// method and a non-null id) and therefore expects a reply.
func expectsResponse(body []byte) bool {
	var m struct {
		ID     json.RawMessage `json:"id"`
		Method *string         `json:"method"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		// Unparseable: forward and wait for a reply rather than hang silently.
		return true
	}
	if m.Method == nil {
		return false
	}
	id := strings.TrimSpace(string(m.ID))
	return id != "" && id != "null"
}

// initializeRequest is the MCP initialize handshake the CLI sends to upstreams.
// clientInfo.version reflects the CLI build version (set via ldflags).
var initializeRequest = fmt.Sprintf(
	`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcpzero","version":%q}}}`,
	version.Version,
)

const initializedNotification = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
