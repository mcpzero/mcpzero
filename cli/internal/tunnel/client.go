package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mcpzero/mcpzero/cli/internal/upstream"
)

const protocolVersion = 2

// NamedUpstream pairs a normalized server name with its upstream. The name is
// the path segment clients use (/v1/<ep>/<name>) and the routing key
// the gateway sends on each mcp_request.
type NamedUpstream struct {
	Name     string
	Upstream upstream.Upstream
}

type Client struct {
	GWBase       string
	EndpointID   string
	Token        string
	RefreshToken string

	// Upstream is a single, unnamed MCP server this tunnel proxies (the legacy
	// single-server form for --mcp-cmd / --mcp-url). Mutually exclusive with
	// Upstreams.
	Upstream upstream.Upstream

	// Upstreams is the ordered set of named MCP servers multiplexed over this
	// one tunnel (the --mcp-config form). Each is addressed by its name.
	Upstreams []NamedUpstream

	// registry maps a server name to its upstream, resolved once in Run. The
	// single-server form registers under the empty-string key.
	registry map[string]upstream.Upstream
	order    []string

	// mu guards curOut, the active connection's outbound queue, used to relay
	// server-initiated upstream events across reconnects.
	mu     sync.Mutex
	curOut chan []byte
}

// DisconnectError describes why the tunnel's websocket ended.
//   - Terminal: the gateway gracefully closed the tunnel (e.g. replaced by a
//     newer connection); treated as a normal stop rather than a failure.
//   - Reconnect: the drop was transient (network) and the tunnel should retry.
//
// A disconnect that is neither Terminal nor Reconnect (e.g. a policy/auth
// rejection) is a fatal error.
type DisconnectError struct {
	Msg       string
	Terminal  bool
	Reconnect bool
}

func (e *DisconnectError) Error() string { return e.Msg }

// retryableError marks a transient network failure that should trigger a
// reconnect attempt (as opposed to a fatal error or a deliberate gateway close).
type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// maxReconnectAttempts bounds how many times the tunnel retries a lost
// connection before giving up.
const maxReconnectAttempts = 30

// reconnectBackoffs is the wait before each reconnect attempt: 1s, 2s, 4s, 8s,
// then reconnectCapBackoff (15s) for every subsequent attempt.
var reconnectBackoffs = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
}

const reconnectCapBackoff = 15 * time.Second

// backoffForAttempt returns the wait before the given 1-based reconnect attempt.
func backoffForAttempt(attempt int) time.Duration {
	if attempt >= 1 && attempt <= len(reconnectBackoffs) {
		return reconnectBackoffs[attempt-1]
	}
	return reconnectCapBackoff
}

type cliRefreshAuth struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// registerServerInfo carries one multiplexed server's name and its concrete
// upstream transport ("stdio" | "streamable-http" | "sse"), so the gateway can
// label each server in the dashboard with its own transport and setup guidance.
type registerServerInfo struct {
	Name      string `json:"name"`
	Transport string `json:"transport"`
}

type registerMessage struct {
	Type            string          `json:"type"`
	EndpointID      string          `json:"endpointId"`
	Token           string          `json:"token,omitempty"`
	Auth            *cliRefreshAuth `json:"auth,omitempty"`
	ProtocolVersion int             `json:"protocolVersion"`
	Transport       string          `json:"transport,omitempty"`
	Capabilities    []string        `json:"capabilities,omitempty"`
	// Servers lists the named MCP servers multiplexed over this tunnel. Empty
	// for a single-server tunnel. Kept as a plain name list for routing and
	// backward compatibility with older gateways.
	Servers []string `json:"servers,omitempty"`
	// ServerInfos carries the same named servers with their per-server
	// transport. Empty for a single-server tunnel (whose transport is the
	// top-level Transport field).
	ServerInfos []registerServerInfo `json:"serverInfos,omitempty"`
}

type mcpRequestMessage struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Body   string `json:"body"`
	Server string `json:"server,omitempty"`
}

type mcpStreamChunkMessage struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Body string `json:"body"`
}

type mcpStreamEndMessage struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Error string `json:"error,omitempty"`
}

type mcpEventMessage struct {
	Type   string `json:"type"`
	Body   string `json:"body"`
	Server string `json:"server,omitempty"`
}

type mcpCancelMessage struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type genericMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (c *Client) Run(ctx context.Context) error {
	if err := c.buildRegistry(); err != nil {
		return err
	}

	wsURL, err := buildTunnelURL(c.GWBase, c.EndpointID)
	if err != nil {
		return err
	}

	// The MCP upstreams start once and stay alive across websocket
	// reconnects; they get their own cancelable context so we can tear them
	// (and, for stdio, their whole process group) down when the tunnel ends.
	mcpCtx, cancelMCP := context.WithCancel(ctx)
	defer cancelMCP()

	for _, name := range c.order {
		up := c.registry[name]
		if err := up.Initialize(mcpCtx); err != nil {
			cancelMCP()
			c.closeAll()
			return fmt.Errorf("initialize server %q: %w", displayName(name), err)
		}
		logEvent("mcp session initialized: %s (%s)", displayName(name), up.Transport())
		go c.forwardEvents(mcpCtx, name, up)
	}

	runErr := c.serveWithReconnect(ctx, wsURL)

	cancelMCP()
	c.closeAll()
	return runErr
}

// buildRegistry resolves the configured upstream(s) into a name->upstream map.
// The single-server form (Upstream) registers under the empty-string key; the
// multi-server form (Upstreams) registers under each server's name.
func (c *Client) buildRegistry() error {
	c.registry = make(map[string]upstream.Upstream)
	c.order = nil

	if len(c.Upstreams) > 0 {
		if c.Upstream != nil {
			return fmt.Errorf("Upstream and Upstreams are mutually exclusive")
		}
		for _, nu := range c.Upstreams {
			if nu.Name == "" {
				return fmt.Errorf("named upstream has an empty name")
			}
			if _, dup := c.registry[nu.Name]; dup {
				return fmt.Errorf("duplicate server name %q", nu.Name)
			}
			c.registry[nu.Name] = nu.Upstream
			c.order = append(c.order, nu.Name)
		}
		return nil
	}

	if c.Upstream != nil {
		c.registry[""] = c.Upstream
		c.order = append(c.order, "")
		return nil
	}

	return fmt.Errorf("no upstream configured")
}

// serverNames returns the named (non-default) servers for the register message.
func (c *Client) serverNames() []string {
	var names []string
	for _, n := range c.order {
		if n != "" {
			names = append(names, n)
		}
	}
	return names
}

// serverInfos returns each named server paired with its concrete upstream
// transport, for the gateway to label per-server in the dashboard. Empty for a
// single-server tunnel (its transport is advertised via primaryTransport).
func (c *Client) serverInfos() []registerServerInfo {
	var infos []registerServerInfo
	for _, n := range c.order {
		if n == "" {
			continue
		}
		infos = append(infos, registerServerInfo{Name: n, Transport: c.registry[n].Transport()})
	}
	return infos
}

// primaryTransport returns the transport hint advertised on register: the sole
// upstream's transport, or the first server's for a multiplexed tunnel.
func (c *Client) primaryTransport() string {
	if len(c.order) == 0 {
		return ""
	}
	return c.registry[c.order[0]].Transport()
}

// resolveUpstream selects the upstream a request targets. A named request must
// match a registered server; an unnamed request resolves to the sole upstream
// when the tunnel proxies exactly one.
func (c *Client) resolveUpstream(server string) (upstream.Upstream, bool) {
	if up, ok := c.registry[server]; ok {
		return up, true
	}
	if server == "" && len(c.registry) == 1 {
		for _, up := range c.registry {
			return up, true
		}
	}
	return nil, false
}

// closeAll releases every registered upstream.
func (c *Client) closeAll() {
	for _, up := range c.registry {
		_ = up.Close()
	}
}

// forwardEvents relays a single upstream's server-initiated messages to the
// gateway over whichever connection is currently active, tagged with the
// originating server name.
func (c *Client) forwardEvents(ctx context.Context, server string, up upstream.Upstream) {
	events := up.Events()
	if events == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-events:
			if !ok {
				return
			}
			c.sendActive(mustMarshal(mcpEventMessage{Type: "mcp_event", Server: server, Body: string(msg)}))
		}
	}
}

// displayName renders a server name for logs, mapping the default (unnamed)
// upstream to "default".
func displayName(name string) string {
	if name == "" {
		return "default"
	}
	return name
}

// serveWithReconnect establishes the websocket, serves traffic, and—on a
// network disconnect—keeps retrying the connection (with backoff) until it
// reconnects or the tunnel is stopped. A deliberate gateway close (e.g. the
// tunnel being replaced) or a local MCP failure ends the loop instead.
func (c *Client) serveWithReconnect(ctx context.Context, wsURL string) error {
	// Initial connection fails fast: a bad endpoint/token/host should surface
	// immediately rather than spin forever at startup.
	conn, err := c.dialAndRegister(ctx, wsURL)
	if err != nil {
		return err
	}

	for {
		err = c.serve(ctx, conn)
		_ = conn.Close()

		if ctx.Err() != nil {
			logEvent("shutdown requested; closing tunnel")
			return nil
		}

		var de *DisconnectError
		if errors.As(err, &de) && de.Terminal {
			// Gateway deliberately ended the tunnel (e.g. replaced): don't
			// reconnect.
			return err
		}
		if !shouldReconnect(err) {
			// Fatal (e.g. local MCP process failed): reconnecting won't help.
			return err
		}

		logEvent("connection lost; attempting to reconnect")
		conn, err = c.reconnect(ctx, wsURL)
		if err != nil {
			// A non-retryable rejection happened while reconnecting.
			return err
		}
		if conn == nil {
			// ctx canceled while waiting to reconnect (tunnel stopped).
			logEvent("shutdown requested; closing tunnel")
			return nil
		}
	}
}

// reconnect retries dial+register on the fixed backoff schedule until it
// succeeds, hits a non-retryable error, exhausts maxReconnectAttempts, or ctx
// is canceled (returns nil, nil).
func (c *Client) reconnect(ctx context.Context, wsURL string) (*websocket.Conn, error) {
	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		wait := backoffForAttempt(attempt)
		logEvent("reconnecting in %s (attempt %d/%d)…", wait, attempt, maxReconnectAttempts)
		if !sleepCtx(ctx, wait) {
			return nil, nil
		}

		conn, err := c.dialAndRegister(ctx, wsURL)
		if err == nil {
			logEvent("reconnected to gateway after %d attempt(s)", attempt)
			return conn, nil
		}
		if ctx.Err() != nil {
			return nil, nil
		}
		if !shouldReconnect(err) {
			return nil, err
		}

		logEvent("reconnect attempt %d/%d failed: %v", attempt, maxReconnectAttempts, err)
	}

	return nil, fmt.Errorf("gave up reconnecting after %d attempts", maxReconnectAttempts)
}

// serve runs the read, write, and ping loops over a single connection,
// returning when any loop ends. All goroutines are stopped before it returns.
func (c *Client) serve(ctx context.Context, conn *websocket.Conn) error {
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make(chan []byte, 64)
	c.setActive(out)
	defer c.clearActive(out)

	// Record a gateway-sent close frame (code + reason) the instant it arrives.
	// A deliberate kick (e.g. "replaced_by_new_connection", code 1000) must be
	// classified from this frame, not from whichever loop happens to fail first.
	// Otherwise a concurrent write/ping error on the closing socket could be
	// mistaken for a retryable network drop, making the kicked tunnel reconnect
	// and evict the connection that replaced it — an endless mutual kick.
	var closeMu sync.Mutex
	var gatewayClose *websocket.CloseError
	conn.SetCloseHandler(func(code int, text string) error {
		closeMu.Lock()
		gatewayClose = &websocket.CloseError{Code: code, Text: text}
		closeMu.Unlock()
		// Mirror gorilla's default handler so the gateway sees our close reply.
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(code, ""),
			time.Now().Add(time.Second),
		)
		return nil
	})

	// Closing the conn when the per-connection context is canceled unblocks a
	// blocked ReadMessage (covers both parent shutdown and a sibling failing).
	go func() {
		<-connCtx.Done()
		_ = conn.Close()
	}()

	errCh := make(chan error, 3)
	go func() { errCh <- c.writeLoop(connCtx, conn, out) }()
	go func() { errCh <- c.readLoop(connCtx, conn, out) }()
	go func() { errCh <- c.pingLoop(connCtx, out) }()

	err := <-errCh
	cancel()
	<-errCh // wait for the remaining goroutines to exit
	<-errCh

	// A close frame from the gateway is the authoritative reason for the
	// disconnect, overriding any racing write/ping error.
	closeMu.Lock()
	gc := gatewayClose
	closeMu.Unlock()
	if gc != nil {
		var de *DisconnectError
		if errors.As(err, &de) && de.Terminal {
			// The read loop already observed and logged this close.
			return err
		}
		reason := disconnectReason(gc)
		logEvent("%s", reason)
		return classifyDisconnect(gc, reason)
	}
	return err
}

func (c *Client) dialAndRegister(ctx context.Context, wsURL string) (*websocket.Conn, error) {
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			e := fmt.Errorf("websocket dial failed (HTTP %d): %w", resp.StatusCode, err)
			// 4xx (other than 408/429) are client errors that won't fix
			// themselves; everything else is treated as transient.
			if resp.StatusCode >= 400 && resp.StatusCode < 500 &&
				resp.StatusCode != 408 && resp.StatusCode != 429 {
				return nil, e
			}
			return nil, &retryableError{e}
		}
		return nil, &retryableError{fmt.Errorf("websocket dial: %w", err)}
	}

	if err := c.register(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// shouldReconnect reports whether an error from serve/dialAndRegister is a
// transient network failure that warrants a reconnect.
func shouldReconnect(err error) bool {
	var de *DisconnectError
	if errors.As(err, &de) {
		return de.Reconnect
	}
	var re *retryableError
	return errors.As(err, &re)
}

// sleepCtx sleeps for d unless ctx is canceled first. It returns false if ctx
// was canceled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func buildTunnelURL(gwBase, endpointID string) (string, error) {
	u, err := url.Parse(gwBase)
	if err != nil {
		return "", fmt.Errorf("parse gw base: %w", err)
	}
	scheme, ok := map[string]string{"http": "ws", "https": "wss"}[u.Scheme]
	if !ok {
		return "", fmt.Errorf("unsupported gw scheme: %s", u.Scheme)
	}
	u.Scheme = scheme
	u.Path = fmt.Sprintf("/tunnel/%s", endpointID)
	return u.String(), nil
}

func (c *Client) register(conn *websocket.Conn) error {
	msg := registerMessage{
		Type:            "register",
		EndpointID:      c.EndpointID,
		ProtocolVersion: protocolVersion,
		Transport:       c.primaryTransport(),
		Servers:         c.serverNames(),
		ServerInfos:     c.serverInfos(),
		Capabilities:    []string{"streaming"},
	}
	if c.RefreshToken != "" {
		msg.Auth = &cliRefreshAuth{Type: "cli_refresh", Token: c.RefreshToken}
	} else {
		msg.Token = c.Token
	}

	if err := conn.WriteJSON(msg); err != nil {
		return &retryableError{fmt.Errorf("send register: %w", err)}
	}

	_ = conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, data, err := conn.ReadMessage()
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		return &retryableError{fmt.Errorf("read register response: %w", err)}
	}

	var resp genericMessage
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parse register response: %w", err)
	}

	switch resp.Type {
	case "register_ok":
		logEvent("tunnel registered for endpoint %s", c.EndpointID)
		return nil
	case "error":
		return fmt.Errorf("register rejected: %s", resp.Message)
	default:
		return fmt.Errorf("unexpected register response: %s", string(data))
	}
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn, out chan []byte) error {
	handlers := make(map[string]context.CancelFunc)
	var hmu sync.Mutex
	var wg sync.WaitGroup

	defer func() {
		hmu.Lock()
		for _, cancel := range handlers {
			cancel()
		}
		hmu.Unlock()
		wg.Wait()
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			// Clean shutdown (Ctrl-C / stop) closes the conn from serve; don't
			// report that as an unexpected disconnect.
			if ctx.Err() != nil {
				return ctx.Err()
			}
			reason := disconnectReason(err)
			logEvent("%s", reason)
			return classifyDisconnect(err, reason)
		}

		var envelope genericMessage
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}

		switch envelope.Type {
		case "mcp_request":
			var req mcpRequestMessage
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			reqCtx, cancel := context.WithCancel(ctx)
			hmu.Lock()
			handlers[req.ID] = cancel
			hmu.Unlock()
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					hmu.Lock()
					delete(handlers, req.ID)
					hmu.Unlock()
					cancel()
				}()
				c.handleRequest(reqCtx, out, req)
			}()
		case "mcp_cancel":
			var cm mcpCancelMessage
			if err := json.Unmarshal(data, &cm); err != nil {
				continue
			}
			hmu.Lock()
			if cancel, ok := handlers[cm.ID]; ok {
				cancel()
			}
			hmu.Unlock()
		case "pong":
			continue
		case "error":
			logEvent("gateway error: %s", envelope.Message)
			return fmt.Errorf("gateway error: %s", envelope.Message)
		}
	}
}

// handleRequest dispatches one request to the upstream, streaming each response
// message back as an mcp_stream_chunk and finishing with mcp_stream_end.
func (c *Client) handleRequest(ctx context.Context, out chan []byte, req mcpRequestMessage) {
	emit := func(msg upstream.Message) error {
		return c.sendCtx(ctx, out, mustMarshal(mcpStreamChunkMessage{
			Type: "mcp_stream_chunk",
			ID:   req.ID,
			Body: string(msg),
		}))
	}

	end := mcpStreamEndMessage{Type: "mcp_stream_end", ID: req.ID}

	up, ok := c.resolveUpstream(req.Server)
	if !ok {
		end.Error = fmt.Sprintf("unknown server %q", req.Server)
		logEvent("mcp request %s rejected: unknown server %q", req.ID, req.Server)
		_ = c.sendCtx(ctx, out, mustMarshal(end))
		return
	}

	err := up.Handle(ctx, []byte(req.Body), emit)
	if err != nil && ctx.Err() == nil {
		end.Error = err.Error()
		logEvent("mcp request %s failed: %v", req.ID, err)
	}
	_ = c.sendCtx(ctx, out, mustMarshal(end))
}

func (c *Client) writeLoop(ctx context.Context, conn *websocket.Conn, out chan []byte) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-out:
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return &retryableError{fmt.Errorf("write message: %w", err)}
			}
		}
	}
}

func (c *Client) pingLoop(ctx context.Context, out chan []byte) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	ping := []byte(`{"type":"ping"}`)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.sendCtx(ctx, out, ping); err != nil {
				return err
			}
		}
	}
}

// sendCtx enqueues a pre-marshaled message for the writer, respecting ctx.
func (c *Client) sendCtx(ctx context.Context, out chan []byte, msg []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- msg:
		return nil
	}
}

func (c *Client) setActive(out chan []byte) {
	c.mu.Lock()
	c.curOut = out
	c.mu.Unlock()
}

func (c *Client) clearActive(out chan []byte) {
	c.mu.Lock()
	if c.curOut == out {
		c.curOut = nil
	}
	c.mu.Unlock()
}

// sendActive best-effort delivers a message over the active connection,
// dropping it if there is no connection or the queue is full.
func (c *Client) sendActive(msg []byte) {
	c.mu.Lock()
	out := c.curOut
	c.mu.Unlock()
	if out == nil {
		return
	}
	select {
	case out <- msg:
	default:
	}
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		// All message types here marshal cleanly; this should never happen.
		return []byte(`{"type":"error","message":"marshal_failed"}`)
	}
	return data
}

// logEvent writes a timestamped lifecycle/diagnostic line to stderr so that
// connection events are visible in the foreground output and the background
// tunnel log file.
func logEvent(format string, args ...any) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] "+format+"\n", append([]any{ts}, args...)...)
}

// disconnectReason produces a human-readable explanation for a dropped
// websocket connection, distinguishing a gateway-initiated close (e.g. the
// tunnel being replaced or rejected) from an unexpected network loss.
func disconnectReason(err error) string {
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		reason := strings.TrimSpace(ce.Text)
		switch {
		case ce.Code == websocket.CloseAbnormalClosure || ce.Code == websocket.CloseTLSHandshake:
			// 1006: connection dropped without a proper close frame (network).
			return fmt.Sprintf("tunnel disconnected unexpectedly (network): %v", err)
		case ce.Code == websocket.CloseNormalClosure && reason == "replaced_by_new_connection":
			return "gateway closed tunnel: replaced by a newer connection to this endpoint"
		case ce.Code == websocket.ClosePolicyViolation:
			if reason == "" {
				reason = "policy violation"
			}
			return fmt.Sprintf("gateway rejected tunnel: %s (code %d)", reason, ce.Code)
		case reason != "":
			return fmt.Sprintf("gateway closed tunnel: %s (code %d)", reason, ce.Code)
		default:
			return fmt.Sprintf("gateway closed tunnel (code %d)", ce.Code)
		}
	}
	if websocket.IsUnexpectedCloseError(err) {
		return fmt.Sprintf("tunnel disconnected unexpectedly: %v", err)
	}
	return fmt.Sprintf("tunnel connection lost (network): %v", err)
}

// classifyDisconnect maps a websocket read error to a DisconnectError that
// captures whether the drop is a graceful gateway close (stop), a transient
// network failure (reconnect), or a deliberate rejection (fatal).
func classifyDisconnect(err error, reason string) *DisconnectError {
	var ce *websocket.CloseError
	if errors.As(err, &ce) {
		switch ce.Code {
		case websocket.CloseNormalClosure, websocket.CloseGoingAway:
			// Gateway deliberately ended the tunnel (e.g. replaced).
			return &DisconnectError{Msg: reason, Terminal: true}
		case websocket.CloseAbnormalClosure, websocket.CloseTLSHandshake, websocket.CloseServiceRestart, websocket.CloseTryAgainLater:
			// Network-level drop or temporary unavailability: retry.
			return &DisconnectError{Msg: reason, Reconnect: true}
		default:
			// Policy/auth/protocol rejection: fatal, no reconnect.
			return &DisconnectError{Msg: reason}
		}
	}
	// Non-close read error (e.g. connection reset): treat as network drop.
	return &DisconnectError{Msg: reason, Reconnect: true}
}
