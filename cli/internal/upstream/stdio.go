package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/mcpzero/mcpzero/cli/internal/stdio"
)

// StdioUpstream proxies a local stdio MCP subprocess. A single persistent
// reader goroutine demultiplexes the server's stdout: JSON-RPC responses are
// correlated back to the waiting request by id, while server-initiated requests
// and notifications (e.g. roots/list, sampling/createMessage) are surfaced on
// Events so the tunnel can forward them to the client. This makes the stdio
// transport fully bidirectional, like the HTTP upstream.
type StdioUpstream struct {
	command string
	workDir string
	env     []string
	onStart func(pid int)

	stdin io.WriteCloser
	read  func() ([]byte, error)
	wait  func() error

	// writeMu serializes writes to the subprocess stdin so concurrent Handle
	// calls can't interleave bytes of different messages.
	writeMu sync.Mutex

	// pending maps an in-flight request id to the channel awaiting its
	// response, delivered by the reader loop.
	pendingMu sync.Mutex
	pending   map[string]chan Message

	events    chan Message
	done      chan struct{}
	closeOnce sync.Once
}

// NewStdio creates a stdio upstream. env holds extra KEY=VALUE environment
// entries layered on the parent environment. onStart, if set, is called with
// the subprocess group id once the server has started.
func NewStdio(command, workDir string, env []string, onStart func(pid int)) *StdioUpstream {
	return &StdioUpstream{command: command, workDir: workDir, env: env, onStart: onStart}
}

func (s *StdioUpstream) Transport() string { return "stdio" }

func (s *StdioUpstream) Initialize(ctx context.Context) error {
	bridge := stdio.Bridge{Command: s.command, WorkDir: s.workDir, Env: s.env}
	stdin, read, wait, pid, err := bridge.Run(ctx)
	if err != nil {
		return err
	}
	s.stdin = stdin
	s.read = read
	s.wait = wait
	if s.onStart != nil {
		s.onStart(pid)
	}
	// The handshake runs synchronously before the demux loop starts, so the
	// initialize response is read directly here (no concurrent reader yet).
	if err := stdio.InitializeSession(stdin, read, stdio.DefaultInitTimeout); err != nil {
		return fmt.Errorf("mcp initialize: %w", err)
	}

	s.pending = make(map[string]chan Message)
	s.events = make(chan Message, 16)
	s.done = make(chan struct{})
	go s.readLoop()
	return nil
}

func (s *StdioUpstream) Handle(ctx context.Context, reqBody []byte, emit Emit) error {
	if s.stdin == nil || s.read == nil {
		return fmt.Errorf("stdio upstream not initialized")
	}

	wantsReply := expectsResponse(reqBody)
	id := jsonRPCID(reqBody)

	// Register the waiter before writing so a fast response can't race ahead of
	// the pending entry.
	var replyCh chan Message
	if wantsReply && id != "" {
		replyCh = make(chan Message, 1)
		s.pendingMu.Lock()
		s.pending[id] = replyCh
		s.pendingMu.Unlock()
		defer func() {
			s.pendingMu.Lock()
			delete(s.pending, id)
			s.pendingMu.Unlock()
		}()
	}

	s.writeMu.Lock()
	err := stdio.WriteMessage(s.stdin, reqBody)
	s.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("write to mcp stdin: %w", err)
	}

	// Notifications, and client→server responses (replies to a server-initiated
	// request), get no reply; don't block waiting for one.
	if !wantsReply {
		return nil
	}
	// A request that expects a reply but carries no correlatable id can't be
	// matched by the reader loop; forward it and return rather than hang.
	if replyCh == nil {
		return nil
	}

	select {
	case msg := <-replyCh:
		return emit(msg)
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return fmt.Errorf("stdio upstream closed before response")
	}
}

// readLoop is the single consumer of the subprocess stdout. It runs until the
// process exits (stdout EOF) or Close is called, then closes Events and wakes
// any pending waiters.
func (s *StdioUpstream) readLoop() {
	defer close(s.events)
	for {
		line, err := s.read()
		if err != nil {
			s.signalDone()
			return
		}
		if len(line) == 0 {
			continue
		}
		if hasMethod(line) {
			// Server-initiated request or notification → forward to the client.
			s.deliverEvent(line)
			continue
		}
		// A response: route to the waiting request by id.
		id := jsonRPCID(line)
		if id == "" {
			continue
		}
		s.pendingMu.Lock()
		ch, ok := s.pending[id]
		s.pendingMu.Unlock()
		if ok {
			select {
			case ch <- line:
			default:
			}
		}
	}
}

func (s *StdioUpstream) deliverEvent(msg Message) {
	select {
	case s.events <- msg:
	default:
		// Drop if no listener is keeping up (matches the HTTP upstream).
	}
}

func (s *StdioUpstream) signalDone() {
	if s.done == nil {
		return
	}
	s.closeOnce.Do(func() { close(s.done) })
}

// Events returns server-initiated messages (requests + notifications) read off
// the subprocess stdout.
func (s *StdioUpstream) Events() <-chan Message {
	return s.events
}

func (s *StdioUpstream) Close() error {
	if s.stdin != nil {
		// Closing stdin makes the server exit, which EOFs stdout and ends the
		// reader loop.
		_ = s.stdin.Close()
	}
	s.signalDone()
	if s.wait != nil {
		_ = s.wait()
	}
	return nil
}

// hasMethod reports whether a JSON-RPC message carries a "method" field, i.e.
// it is a request or notification rather than a response.
func hasMethod(body []byte) bool {
	var m struct {
		Method *string `json:"method"`
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return false
	}
	return m.Method != nil
}
