package stdio

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Bridge runs a local MCP server subprocess and reads/writes newline-delimited JSON-RPC messages.
type Bridge struct {
	Command string
	WorkDir string
	// Env holds additional environment variables (KEY=VALUE) layered on top of
	// the parent process environment.
	Env []string
}

// Run starts the MCP subprocess. The returned pid is the process-group leader
// id so the entire subprocess tree (e.g. `sh -c "python ..."` and its
// descendants) can be terminated together. When ctx is canceled the whole
// group is signalled, not just the direct child.
func (b *Bridge) Run(ctx context.Context) (stdin io.WriteCloser, readFn func() ([]byte, error), wait func() error, pid int, err error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", b.Command)
	if b.WorkDir != "" {
		cmd.Dir = b.WorkDir
	}
	if len(b.Env) > 0 {
		cmd.Env = append(os.Environ(), b.Env...)
	}
	// Put the child in its own process group and tear the whole group down
	// on cancellation so no grandchildren are orphaned.
	configureProcAttr(cmd)

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, nil, 0, fmt.Errorf("start mcp process: %w", err)
	}

	// stderr is for logs only; MCP readiness is determined by the initialize handshake.
	go relayStderr(stderrPipe)

	reader := bufio.NewReader(stdoutPipe)
	var readMu sync.Mutex

	readMessage := func() ([]byte, error) {
		readMu.Lock()
		defer readMu.Unlock()
		return readLineMessage(reader)
	}

	return stdinPipe, readMessage, cmd.Wait, cmd.Process.Pid, nil
}

func relayStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintln(os.Stderr, scanner.Text())
	}
}

func WriteMessage(w io.Writer, body []byte) error {
	if !bytes.HasSuffix(body, []byte("\n")) {
		body = append(body, '\n')
	}
	_, err := w.Write(body)
	return err
}

func readLineMessage(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(line), nil
}
