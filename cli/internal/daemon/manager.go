package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/mcpzero/mcpzero/cli/internal/secret"
	"github.com/mcpzero/mcpzero/cli/internal/upstream"
)

// InternalRunCommand is the hidden subcommand the detached child runs.
const InternalRunCommand = "__daemon-run"

// TokenEnvVar passes a raw tunnel token to the detached child without
// persisting it to disk.
const TokenEnvVar = "MCPZERO_DAEMON_TOKEN"

// ServerSpec describes one named MCP server to multiplex over a tunnel. Env and
// MCPHeaders carry resolved plaintext values (encrypted before persisting).
type ServerSpec struct {
	Name         string
	MCPCmd       string
	MCPWorkDir   string
	Env          []upstream.Header // KEY/VALUE pairs
	MCPURL       string
	MCPTransport string
	MCPHeaders   []upstream.Header
}

// SpawnConfig carries everything required to launch a detached tunnel.
type SpawnConfig struct {
	EndpointID string
	MCPCmd     string
	MCPWorkDir string
	// HTTP upstream (alternative to MCPCmd).
	MCPURL       string
	MCPTransport string
	MCPHeaders   []upstream.Header // resolved header values (encrypted before persisting)
	// Servers, when set, multiplexes multiple named MCP servers over one
	// tunnel (the --mcp-config form) instead of the single MCPCmd/MCPURL.
	Servers []ServerSpec
	GWBase  string
	Token   string // raw tunnel token (optional); passed via env, never persisted
}

// Supported reports whether background tunnels work on this platform.
func Supported() bool { return daemonSupported }

// Start launches a detached tunnel process and records its state.
func Start(cfg SpawnConfig) (*State, error) {
	if !daemonSupported {
		return nil, fmt.Errorf("background tunnels are not supported on this platform")
	}

	hash, err := NewHash()
	if err != nil {
		return nil, err
	}

	encHeaders, err := encryptHeaders(cfg.MCPHeaders)
	if err != nil {
		return nil, err
	}

	servers, err := encryptServers(cfg.Servers)
	if err != nil {
		return nil, err
	}

	s := &State{
		Hash:          hash,
		EndpointID:    cfg.EndpointID,
		MCPCmd:        cfg.MCPCmd,
		MCPWorkDir:    cfg.MCPWorkDir,
		MCPURL:        cfg.MCPURL,
		MCPTransport:  cfg.MCPTransport,
		MCPHeadersEnc: encHeaders,
		Servers:       servers,
		GWBase:        cfg.GWBase,
		StartedAt:     time.Now(),
		Status:        StatusStarting,
	}
	if err := Save(s); err != nil {
		return nil, err
	}

	logPath, err := LogPath(hash)
	if err != nil {
		_ = Remove(hash)
		return nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = Remove(hash)
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	exe, err := os.Executable()
	if err != nil {
		_ = Remove(hash)
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(exe, "tunnel", InternalRunCommand, hash)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = detachSysProcAttr()
	cmd.Env = os.Environ()
	if cfg.Token != "" {
		cmd.Env = append(cmd.Env, TokenEnvVar+"="+cfg.Token)
	}

	if err := cmd.Start(); err != nil {
		_ = Remove(hash)
		return nil, fmt.Errorf("spawn daemon: %w", err)
	}

	s.PID = cmd.Process.Pid
	s.Status = StatusRunning
	if err := Save(s); err != nil {
		return nil, err
	}

	// Detach: the child keeps running after the parent exits.
	_ = cmd.Process.Release()
	return s, nil
}

// encryptHeaders encrypts each header value for at-rest storage.
func encryptHeaders(headers []upstream.Header) ([]EncHeader, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	out := make([]EncHeader, 0, len(headers))
	for _, h := range headers {
		enc, err := secret.Encrypt(h.Value)
		if err != nil {
			return nil, fmt.Errorf("encrypt header %q: %w", h.Name, err)
		}
		out = append(out, EncHeader{Name: h.Name, ValueEnc: enc})
	}
	return out, nil
}

// Headers decrypts the stored HTTP upstream headers for this tunnel.
func (s *State) Headers() ([]upstream.Header, error) {
	return decryptKVs(s.MCPHeadersEnc)
}

// encryptServers encrypts the env and header values of each named server.
func encryptServers(servers []ServerSpec) ([]ServerState, error) {
	if len(servers) == 0 {
		return nil, nil
	}
	out := make([]ServerState, 0, len(servers))
	for _, sv := range servers {
		envEnc, err := encryptHeaders(sv.Env)
		if err != nil {
			return nil, fmt.Errorf("encrypt env for server %q: %w", sv.Name, err)
		}
		headersEnc, err := encryptHeaders(sv.MCPHeaders)
		if err != nil {
			return nil, fmt.Errorf("encrypt headers for server %q: %w", sv.Name, err)
		}
		out = append(out, ServerState{
			Name:          sv.Name,
			MCPCmd:        sv.MCPCmd,
			MCPWorkDir:    sv.MCPWorkDir,
			EnvEnc:        envEnc,
			MCPURL:        sv.MCPURL,
			MCPTransport:  sv.MCPTransport,
			MCPHeadersEnc: headersEnc,
		})
	}
	return out, nil
}

// Env decrypts the stored environment variables for a named server.
func (s *ServerState) Env() ([]upstream.Header, error) {
	return decryptKVs(s.EnvEnc)
}

// Headers decrypts the stored HTTP headers for a named server.
func (s *ServerState) Headers() ([]upstream.Header, error) {
	return decryptKVs(s.MCPHeadersEnc)
}

// decryptKVs decrypts a list of encrypted name/value pairs.
func decryptKVs(enc []EncHeader) ([]upstream.Header, error) {
	if len(enc) == 0 {
		return nil, nil
	}
	out := make([]upstream.Header, 0, len(enc))
	for _, h := range enc {
		value, err := secret.Decrypt(h.ValueEnc)
		if err != nil {
			return nil, fmt.Errorf("decrypt %q: %w", h.Name, err)
		}
		out = append(out, upstream.Header{Name: h.Name, Value: value})
	}
	return out, nil
}

// Stop terminates a tunnel's daemon process and its MCP subprocess tree, then
// marks the tunnel stopped. State and log files are kept so `tunnel logs`/`list`
// still work; use Remove to clear them.
func Stop(s *State) error {
	if processAlive(s.PID) {
		// SIGTERM lets the daemon gracefully tear down its MCP process group.
		_ = signalStop(s.PID)

		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if !processAlive(s.PID) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if processAlive(s.PID) {
			_ = signalKill(s.PID)
		}
	}

	// Safety net: ensure the MCP subprocess tree is gone even if the daemon
	// was force-killed before it could reap its own children.
	if s.MCPGroupPID > 0 {
		_ = killGroup(s.MCPGroupPID)
	}
	for _, sv := range s.Servers {
		if sv.MCPGroupPID > 0 {
			_ = killGroup(sv.MCPGroupPID)
		}
	}

	return MarkStatus(s.Hash, StatusStopped)
}
