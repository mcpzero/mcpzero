package mcpconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CursorAddOptions configures writing an HTTP MCP server entry into Cursor's mcp.json.
type CursorAddOptions struct {
	ConfigPath string // absolute path; when empty, resolved from Project + WorkDir
	WorkDir    string
	Project    bool
	ServerName string
	URL        string
	Headers    map[string]string
}

type httpServerEntry struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// DefaultCursorConfigPath returns the global or project Cursor MCP config path.
func DefaultCursorConfigPath(project bool, workDir string) (string, error) {
	if project {
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("resolve working directory: %w", err)
			}
		}
		return filepath.Join(workDir, ".cursor", "mcp.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".cursor", "mcp.json"), nil
}

// GatewayEndpointURL builds the MCP gateway URL for an endpoint (optionally a direct server route).
func GatewayEndpointURL(gwBase, endpointID, server string) string {
	base := strings.TrimRight(gwBase, "/")
	path := "/v1/" + strings.TrimPrefix(endpointID, "/")
	if server != "" {
		path = path + "/" + strings.Trim(server, "/")
	}
	return base + path
}

// AddCursorHTTPServer merges an HTTP MCP server entry into Cursor's mcp.json.
func AddCursorHTTPServer(opts CursorAddOptions) (string, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		var err error
		configPath, err = DefaultCursorConfigPath(opts.Project, opts.WorkDir)
		if err != nil {
			return "", err
		}
	}

	name := strings.TrimSpace(opts.ServerName)
	if name == "" {
		name = "mcpzero"
	}
	if strings.TrimSpace(opts.URL) == "" {
		return "", fmt.Errorf("url is required")
	}

	top := map[string]json.RawMessage{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &top); err != nil {
			return "", fmt.Errorf("parse %s: %w", configPath, err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", configPath, err)
	}

	serversKey := "mcpServers"
	servers := map[string]json.RawMessage{}
	if raw, ok := top[serversKey]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return "", fmt.Errorf("parse %s mcpServers: %w", configPath, err)
		}
	}

	entry := httpServerEntry{
		URL:     opts.URL,
		Headers: copyHeaders(opts.Headers),
	}
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	servers[name] = entryJSON

	serversJSON, err := json.Marshal(servers)
	if err != nil {
		return "", err
	}
	top[serversKey] = serversJSON

	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return "", err
	}
	out = append(out, '\n')

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", configPath, err)
	}

	return configPath, nil
}

func copyHeaders(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
