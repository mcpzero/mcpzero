package upstream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Transport modes for an HTTP upstream.
const (
	TransportAuto       = "auto"
	TransportStreamable = "streamable-http"
	TransportSSE        = "sse"
)

// HTTPUpstream proxies an HTTP MCP server. It speaks the Streamable HTTP
// transport (POST returning JSON or an SSE stream, plus an optional GET stream
// for server-initiated messages) and the legacy HTTP+SSE transport.
type HTTPUpstream struct {
	rawURL    string
	headers   []Header
	transport string
	client    *http.Client

	mu        sync.Mutex
	sessionID string
	// resolved is the concrete wire transport ("streamable-http" | "sse")
	// determined once Initialize connects. For "auto" this records which path
	// actually succeeded so register can advertise the real transport.
	resolved string

	events chan Message

	// Legacy SSE state.
	messageURL string
	pending    map[string]chan Message
	pendingMu  sync.Mutex
	connected  chan struct{} // closed once the legacy SSE endpoint is known
}

// NewHTTP creates an HTTP upstream. transport is one of auto|streamable-http|sse.
func NewHTTP(rawURL string, headers []Header, transport string) (*HTTPUpstream, error) {
	if _, err := url.Parse(rawURL); err != nil {
		return nil, fmt.Errorf("invalid --mcp-url: %w", err)
	}
	if transport == "" {
		transport = TransportAuto
	}
	switch transport {
	case TransportAuto, TransportStreamable, TransportSSE:
	default:
		return nil, fmt.Errorf("invalid --mcp-transport %q (auto|streamable-http|sse)", transport)
	}
	return &HTTPUpstream{
		rawURL:    rawURL,
		headers:   headers,
		transport: transport,
		client:    &http.Client{}, // no global timeout: SSE streams are long-lived
		events:    make(chan Message, 16),
		pending:   make(map[string]chan Message),
		connected: make(chan struct{}),
	}, nil
}

// Transport reports the concrete wire transport: "streamable-http" or "sse".
// Before Initialize resolves it, the configured mode is used (with "auto"
// reported as "streamable-http", the path "auto" attempts first).
func (h *HTTPUpstream) Transport() string {
	h.mu.Lock()
	resolved := h.resolved
	h.mu.Unlock()
	if resolved != "" {
		return resolved
	}
	if h.transport == TransportSSE {
		return TransportSSE
	}
	return TransportStreamable
}

func (h *HTTPUpstream) Events() <-chan Message { return h.events }

func (h *HTTPUpstream) setResolved(transport string) {
	h.mu.Lock()
	h.resolved = transport
	h.mu.Unlock()
}

func (h *HTTPUpstream) Initialize(ctx context.Context) error {
	if h.transport == TransportSSE {
		if err := h.initLegacySSE(ctx); err != nil {
			return err
		}
		h.setResolved(TransportSSE)
		return nil
	}

	// Streamable HTTP: best-effort initialize to capture the session id. A
	// plain JSON-RPC HTTP endpoint that doesn't implement initialize will
	// return an error response; that's fine, we still proxy subsequent calls.
	if err := h.Handle(ctx, []byte(initializeRequest), func(Message) error { return nil }); err != nil {
		return fmt.Errorf("connect to %s: %w", h.rawURL, err)
	}
	h.setResolved(TransportStreamable)
	_ = h.Handle(ctx, []byte(initializedNotification), func(Message) error { return nil })

	// Best-effort GET stream for server-initiated messages.
	go h.streamServerEvents()
	return nil
}

func (h *HTTPUpstream) Handle(ctx context.Context, reqBody []byte, emit Emit) error {
	if h.transport == TransportSSE {
		return h.handleLegacySSE(ctx, reqBody, emit)
	}
	return h.handleStreamable(ctx, reqBody, emit)
}

func (h *HTTPUpstream) Close() error {
	h.client.CloseIdleConnections()
	return nil
}

// --- Streamable HTTP ---

func (h *HTTPUpstream) handleStreamable(ctx context.Context, reqBody []byte, emit Emit) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.rawURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	h.applyHeaders(req)
	applyLoopHeader(ctx, req)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	h.captureSession(resp)

	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") {
		return parseSSEStream(ctx, resp.Body, func(event, data string) error {
			if data == "" {
				return nil
			}
			return emit([]byte(data))
		})
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}
	return emit(body)
}

func (h *HTTPUpstream) streamServerEvents() {
	backoff := time.Second
	for {
		err := h.openServerStream()
		if err == nil {
			return // context canceled / clean stop
		}
		time.Sleep(backoff)
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func (h *HTTPUpstream) openServerStream() error {
	req, err := http.NewRequest(http.MethodGet, h.rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	h.applyHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Servers without a GET stream return 405/404: stop trying.
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server event stream status %d", resp.StatusCode)
	}

	return parseSSEStream(context.Background(), resp.Body, func(event, data string) error {
		if data != "" {
			h.deliverEvent([]byte(data))
		}
		return nil
	})
}

// --- Legacy HTTP + SSE ---

func (h *HTTPUpstream) initLegacySSE(ctx context.Context) error {
	ready := make(chan error, 1)
	go h.runLegacyStream(ready)

	select {
	case err := <-ready:
		if err != nil {
			return fmt.Errorf("open sse stream %s: %w", h.rawURL, err)
		}
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for sse endpoint event from %s", h.rawURL)
	case <-ctx.Done():
		return ctx.Err()
	}

	// Best-effort MCP handshake over the discovered message endpoint.
	_ = h.Handle(ctx, []byte(initializeRequest), func(Message) error { return nil })
	_ = h.Handle(ctx, []byte(initializedNotification), func(Message) error { return nil })
	return nil
}

// runLegacyStream opens the persistent GET SSE stream, discovers the message
// endpoint, and routes incoming messages to waiting requests or to events.
func (h *HTTPUpstream) runLegacyStream(ready chan<- error) {
	req, err := http.NewRequest(http.MethodGet, h.rawURL, nil)
	if err != nil {
		ready <- err
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	h.applyHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		ready <- err
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		ready <- fmt.Errorf("status %d", resp.StatusCode)
		return
	}

	signaled := false
	_ = parseSSEStream(context.Background(), resp.Body, func(event, data string) error {
		switch event {
		case "endpoint":
			h.mu.Lock()
			h.messageURL = h.resolveURL(data)
			h.mu.Unlock()
			if !signaled {
				signaled = true
				close(h.connected)
				ready <- nil
			}
		default:
			if data != "" {
				h.routeLegacyMessage([]byte(data))
			}
		}
		return nil
	})
}

func (h *HTTPUpstream) handleLegacySSE(ctx context.Context, reqBody []byte, emit Emit) error {
	select {
	case <-h.connected:
	case <-ctx.Done():
		return ctx.Err()
	}

	h.mu.Lock()
	messageURL := h.messageURL
	h.mu.Unlock()
	if messageURL == "" {
		return fmt.Errorf("sse message endpoint not available")
	}

	wantsReply := expectsResponse(reqBody)
	id := jsonRPCID(reqBody)
	var replyCh chan Message
	if wantsReply && id != "" {
		replyCh = make(chan Message, 1)
		h.pendingMu.Lock()
		h.pending[id] = replyCh
		h.pendingMu.Unlock()
		defer func() {
			h.pendingMu.Lock()
			delete(h.pending, id)
			h.pendingMu.Unlock()
		}()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messageURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	h.applyHeaders(req)
	applyLoopHeader(ctx, req)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	// The reply arrives on the SSE stream, not this POST response.
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if !wantsReply || replyCh == nil {
		return nil
	}

	// Some servers echo the reply on the POST response too; accept it.
	if trimmed := bytes.TrimSpace(body); len(trimmed) > 0 && jsonRPCID(trimmed) == id {
		return emit(trimmed)
	}

	select {
	case msg := <-replyCh:
		return emit(msg)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *HTTPUpstream) routeLegacyMessage(msg Message) {
	id := jsonRPCID(msg)
	if id != "" {
		h.pendingMu.Lock()
		ch, ok := h.pending[id]
		h.pendingMu.Unlock()
		if ok {
			select {
			case ch <- msg:
			default:
			}
			return
		}
	}
	h.deliverEvent(msg)
}

// --- shared helpers ---

// applyLoopHeader propagates the gateway forwarding chain (if any) onto an
// outbound request so the gateway can detect loops when this HTTP upstream
// points back at a gateway endpoint.
func applyLoopHeader(ctx context.Context, req *http.Request) {
	if trace := loopTraceFromContext(ctx); trace != "" {
		req.Header.Set(LoopHeader, trace)
	}
}

func (h *HTTPUpstream) applyHeaders(req *http.Request) {
	for _, hd := range h.headers {
		req.Header.Set(hd.Name, hd.Value)
	}
	h.mu.Lock()
	sid := h.sessionID
	h.mu.Unlock()
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
}

func (h *HTTPUpstream) captureSession(resp *http.Response) {
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		h.mu.Lock()
		h.sessionID = sid
		h.mu.Unlock()
	}
}

func (h *HTTPUpstream) deliverEvent(msg Message) {
	select {
	case h.events <- msg:
	default:
		// drop if no listener is keeping up
	}
}

func (h *HTTPUpstream) resolveURL(ref string) string {
	base, err := url.Parse(h.rawURL)
	if err != nil {
		return ref
	}
	rel, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	return base.ResolveReference(rel).String()
}

// parseSSEStream reads an SSE body, invoking onEvent(eventType, data) for each
// event (data is the joined data lines). It returns nil when the stream ends
// or ctx is canceled.
func parseSSEStream(ctx context.Context, body io.Reader, onEvent func(event, data string) error) error {
	reader := bufio.NewReader(body)
	var dataLines []string
	eventType := "message"

	flush := func() error {
		if len(dataLines) == 0 {
			eventType = "message"
			return nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		ev := eventType
		eventType = "message"
		return onEvent(ev, data)
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			switch {
			case trimmed == "":
				if ferr := flush(); ferr != nil {
					return ferr
				}
			case strings.HasPrefix(trimmed, ":"):
				// comment, ignore
			case strings.HasPrefix(trimmed, "data:"):
				dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(trimmed, "data:"), " "))
			case strings.HasPrefix(trimmed, "event:"):
				eventType = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
			}
		}
		if err != nil {
			_ = flush()
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func jsonRPCID(body []byte) string {
	var m struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return ""
	}
	id := strings.TrimSpace(string(m.ID))
	if id == "" || id == "null" {
		return ""
	}
	return id
}
