package stdio

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/mcpzero/mcpzero/cli/internal/version"
)

const DefaultInitTimeout = 30 * time.Second

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeSession performs the MCP lifecycle handshake over stdio.
// Readiness is defined by a valid initialize response, not stderr banners.
func InitializeSession(
	stdin io.Writer,
	readMessage func() ([]byte, error),
	timeout time.Duration,
) error {
	if timeout <= 0 {
		timeout = DefaultInitTimeout
	}

	initReq := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcpzero","version":%q}}}`,
		version.Version,
	)
	if err := WriteMessage(stdin, []byte(initReq)); err != nil {
		return fmt.Errorf("send initialize: %w", err)
	}

	respCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		// Skip any server-initiated notifications/requests (lines carrying a
		// "method") that arrive before the initialize response, so a startup
		// banner or log notification can't be mistaken for the handshake reply.
		for {
			body, err := readMessage()
			if err != nil {
				errCh <- err
				return
			}
			if isResponse(body) {
				respCh <- body
				return
			}
		}
	}()

	var resp jsonRPCResponse
	select {
	case body := <-respCh:
		if err := json.Unmarshal(body, &resp); err != nil {
			return fmt.Errorf("invalid initialize response (expected JSON-RPC on stdout): %s", truncate(string(body), 160))
		}
	case err := <-errCh:
		return fmt.Errorf("read initialize response: %w", err)
	case <-time.After(timeout):
		return fmt.Errorf(
			"timeout waiting for MCP initialize response (%s); MCP stdio servers must reply to initialize before other requests",
			timeout,
		)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize rejected: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}
	if len(resp.Result) == 0 {
		return fmt.Errorf("initialize response missing result")
	}

	initialized := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	if err := WriteMessage(stdin, []byte(initialized)); err != nil {
		return fmt.Errorf("send notifications/initialized: %w", err)
	}
	return nil
}

// isResponse reports whether a JSON-RPC line is a response (no "method"),
// as opposed to a server-initiated request or notification.
func isResponse(body []byte) bool {
	var m struct {
		Method *string `json:"method"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		// Unparseable: treat as the response so the handshake surfaces a clear
		// "invalid initialize response" error rather than spinning.
		return true
	}
	return m.Method == nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
