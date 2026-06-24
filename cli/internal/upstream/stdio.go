package upstream

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/mcpzero/mcpzero/cli/internal/stdio"
)

// StdioUpstream proxies a local stdio MCP subprocess. Because a single
// stdin/stdout pair cannot multiplex concurrent requests, Handle is serialized:
// each request writes to stdin and reads exactly one response line.
type StdioUpstream struct {
	command string
	workDir string
	env     []string
	onStart func(pid int)

	mu    sync.Mutex
	stdin io.WriteCloser
	read  func() ([]byte, error)
	wait  func() error
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
	if err := stdio.InitializeSession(stdin, read, stdio.DefaultInitTimeout); err != nil {
		return fmt.Errorf("mcp initialize: %w", err)
	}
	return nil
}

func (s *StdioUpstream) Handle(ctx context.Context, reqBody []byte, emit Emit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stdin == nil || s.read == nil {
		return fmt.Errorf("stdio upstream not initialized")
	}

	if err := stdio.WriteMessage(s.stdin, reqBody); err != nil {
		return fmt.Errorf("write to mcp stdin: %w", err)
	}

	// Notifications get no reply; don't block reading one.
	if !expectsResponse(reqBody) {
		return nil
	}

	respBody, err := s.read()
	if err != nil {
		return fmt.Errorf("read from mcp stdout: %w", err)
	}
	return emit(respBody)
}

// Events returns nil: stdio responses are correlated synchronously and the
// current bridge does not surface unsolicited server messages.
func (s *StdioUpstream) Events() <-chan Message { return nil }

func (s *StdioUpstream) Close() error {
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.wait != nil {
		_ = s.wait()
	}
	return nil
}
