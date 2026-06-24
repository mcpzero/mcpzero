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
		body, err := readMessage()
		if err != nil {
			errCh <- err
			return
		}
		respCh <- body
	}()

	var body []byte
	select {
	case body = <-respCh:
	case err := <-errCh:
		return fmt.Errorf("read initialize response: %w", err)
	case <-time.After(timeout):
		return fmt.Errorf(
			"timeout waiting for MCP initialize response (%s); MCP stdio servers must reply to initialize before other requests",
			timeout,
		)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("invalid initialize response (expected JSON-RPC on stdout): %s", truncate(string(body), 160))
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
