package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Status values for a managed tunnel.
const (
	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusStopped  = "stopped"
	StatusError    = "error"
)

// EncHeader is an HTTP upstream header (or env var) whose value is encrypted at
// rest (AES-256-GCM via internal/secret).
type EncHeader struct {
	Name     string `json:"name"`
	ValueEnc string `json:"value_enc"`
}

// ServerState is the persisted metadata for one named MCP server multiplexed
// over a tunnel (the --mcp-config form). Sensitive values (env, headers) are
// encrypted at rest.
type ServerState struct {
	Name        string `json:"name"`
	MCPGroupPID int    `json:"mcp_group_pid,omitempty"`
	// Stdio upstream.
	MCPCmd     string      `json:"mcp_cmd,omitempty"`
	MCPWorkDir string      `json:"mcp_workdir,omitempty"`
	EnvEnc     []EncHeader `json:"env_enc,omitempty"`
	// HTTP upstream (alternative to MCPCmd).
	MCPURL        string      `json:"mcp_url,omitempty"`
	MCPTransport  string      `json:"mcp_transport,omitempty"`
	MCPHeadersEnc []EncHeader `json:"mcp_headers_enc,omitempty"`
}

// State is the persisted metadata for one background tunnel.
type State struct {
	Hash        string    `json:"hash"`
	PID         int       `json:"pid"`
	MCPGroupPID int       `json:"mcp_group_pid"`
	EndpointID  string    `json:"endpoint_id"`
	MCPCmd      string    `json:"mcp_cmd"`
	MCPWorkDir  string    `json:"mcp_workdir"`
	// HTTP upstream (alternative to MCPCmd).
	MCPURL        string      `json:"mcp_url,omitempty"`
	MCPTransport  string      `json:"mcp_transport,omitempty"`
	MCPHeadersEnc []EncHeader `json:"mcp_headers_enc,omitempty"`
	// Servers holds multiple named MCP servers multiplexed over one tunnel
	// (the --mcp-config form). When set, the single MCPCmd/MCPURL fields above
	// are unused.
	Servers   []ServerState `json:"servers,omitempty"`
	GWBase    string        `json:"gw_base"`
	StartedAt time.Time     `json:"started_at"`
	Status    string        `json:"status"`
}

// Target returns a short, human-readable description of what this tunnel
// proxies, for `tunnel list` and conflict messages.
func (s *State) Target() string {
	if len(s.Servers) > 0 {
		names := make([]string, 0, len(s.Servers))
		for _, sv := range s.Servers {
			names = append(names, sv.Name)
		}
		return fmt.Sprintf("%d servers: %s", len(s.Servers), strings.Join(names, ", "))
	}
	if s.MCPCmd != "" {
		return s.MCPCmd
	}
	return s.MCPURL
}

func tunnelsDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(base, "mcpzero", "tunnels"), nil
}

func ensureDir() (string, error) {
	dir, err := tunnelsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create tunnels dir: %w", err)
	}
	return dir, nil
}

func statePath(hash string) (string, error) {
	dir, err := tunnelsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, hash+".json"), nil
}

// LogPath returns the log file path for a tunnel hash.
func LogPath(hash string) (string, error) {
	dir, err := tunnelsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, hash+".log"), nil
}

// NewHash returns a random 8-character hex id.
func NewHash() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate hash: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// Save writes tunnel state to disk.
func Save(s *State) error {
	if _, err := ensureDir(); err != nil {
		return err
	}
	path, err := statePath(s.Hash)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

// Load reads tunnel state by exact hash.
func Load(hash string) (*State, error) {
	path, err := statePath(hash)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no tunnel with id %q", hash)
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &s, nil
}

// Resolve finds a tunnel by exact hash or unique prefix.
func Resolve(prefix string) (*State, error) {
	if prefix == "" {
		return nil, fmt.Errorf("tunnel id is required")
	}
	if s, err := Load(prefix); err == nil {
		return s, nil
	}

	all, err := List()
	if err != nil {
		return nil, err
	}
	var matches []*State
	for _, s := range all {
		if strings.HasPrefix(s.Hash, prefix) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nil, fmt.Errorf("no tunnel with id %q", prefix)
	default:
		return nil, fmt.Errorf("ambiguous id %q matches %d tunnels", prefix, len(matches))
	}
}

// Remove deletes the state and log files for a tunnel.
func Remove(hash string) error {
	path, err := statePath(hash)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state: %w", err)
	}
	if logPath, err := LogPath(hash); err == nil {
		_ = os.Remove(logPath)
	}
	return nil
}

// List returns all known tunnels sorted by start time.
func List() ([]*State, error) {
	dir, err := tunnelsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tunnels dir: %w", err)
	}

	var states []*State
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		hash := strings.TrimSuffix(e.Name(), ".json")
		s, err := Load(hash)
		if err != nil {
			continue
		}
		states = append(states, s)
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].StartedAt.Before(states[j].StartedAt)
	})
	return states, nil
}

// MarkStatus updates the status field of a stored tunnel.
func MarkStatus(hash, status string) error {
	s, err := Load(hash)
	if err != nil {
		return err
	}
	s.Status = status
	return Save(s)
}

// EffectiveStatus reconciles stored status against process liveness.
func EffectiveStatus(s *State) string {
	if s.Status == StatusRunning && !processAlive(s.PID) {
		return StatusStopped
	}
	return s.Status
}

// IsAlive reports whether the tunnel's process is currently running.
func IsAlive(s *State) bool {
	if s == nil {
		return false
	}
	return processAlive(s.PID)
}
