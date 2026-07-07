package mcpconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGatewayEndpointURL(t *testing.T) {
	cases := map[string]string{
		"root":    "https://gw.mcpzero.io/v1/ep_abc",
		"server":  "https://gw.mcpzero.io/v1/ep_abc/filesystem",
		"trimmed": "https://gw.mcpzero.io/v1/ep_abc/nested",
	}
	if got := GatewayEndpointURL("https://gw.mcpzero.io/", "ep_abc", ""); got != cases["root"] {
		t.Fatalf("root = %q", got)
	}
	if got := GatewayEndpointURL("https://gw.mcpzero.io", "ep_abc", "filesystem"); got != cases["server"] {
		t.Fatalf("server = %q", got)
	}
	if got := GatewayEndpointURL("https://gw.mcpzero.io", "/ep_abc", "/nested/"); got != cases["trimmed"] {
		t.Fatalf("trimmed = %q", got)
	}
}

func TestAddCursorHTTPServerMerge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cursor", "mcp.json")

	existing := map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]string{"command": "echo hi"},
		},
	}
	data, _ := json.Marshal(existing)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	gotPath, err := AddCursorHTTPServer(CursorAddOptions{
		ConfigPath: path,
		ServerName: "mcpzero",
		URL:        "https://gw.mcpzero.io/v1/ep_test",
		Headers: map[string]string{
			"Authorization": "Bearer mz_live_test",
		},
	})
	if err != nil {
		t.Fatalf("AddCursorHTTPServer: %v", err)
	}
	if gotPath != path {
		t.Fatalf("path = %q", gotPath)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}

	var servers map[string]json.RawMessage
	if err := json.Unmarshal(parsed["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}

	var other map[string]string
	if err := json.Unmarshal(servers["other"], &other); err != nil {
		t.Fatal(err)
	}
	if other["command"] != "echo hi" {
		t.Fatalf("other server lost: %#v", other)
	}

	var mcpzero httpServerEntry
	if err := json.Unmarshal(servers["mcpzero"], &mcpzero); err != nil {
		t.Fatal(err)
	}
	if mcpzero.URL != "https://gw.mcpzero.io/v1/ep_test" {
		t.Fatalf("url = %q", mcpzero.URL)
	}
	if mcpzero.Headers["Authorization"] != "Bearer mz_live_test" {
		t.Fatalf("headers = %#v", mcpzero.Headers)
	}
}

func TestDefaultCursorConfigPath(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Setenv("HOME", home)
	defer os.Setenv("HOME", oldHome)

	global, err := DefaultCursorConfigPath(false, "")
	if err != nil {
		t.Fatal(err)
	}
	if global != filepath.Join(home, ".cursor", "mcp.json") {
		t.Fatalf("global = %q", global)
	}

	work := t.TempDir()
	project, err := DefaultCursorConfigPath(true, work)
	if err != nil {
		t.Fatal(err)
	}
	if project != filepath.Join(work, ".cursor", "mcp.json") {
		t.Fatalf("project = %q", project)
	}
}
