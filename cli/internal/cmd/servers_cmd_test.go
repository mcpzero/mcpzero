package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunServersRequiresFlag(t *testing.T) {
	err := runServers(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestRunServersListsConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	content := `{
  "mcpServers": {
    "fs": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"] },
    "remote": { "url": "https://example.com/mcp", "type": "http" }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := runServers([]string{"--mcp-config", path})
	_ = w.Close()
	os.Stdout = old
	if runErr != nil {
		t.Fatal(runErr)
	}
	outBytes, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatal(readErr)
	}
	out := string(outBytes)
	if !strings.Contains(out, "fs") || !strings.Contains(out, "remote") {
		t.Fatalf("expected server names in output: %q", out)
	}
	if !strings.Contains(out, "stdio") || !strings.Contains(out, "http") {
		t.Fatalf("expected transports in output: %q", out)
	}
}

func TestRunServersLoopCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	content := `{
  "mcpServers": {
    "loop": { "url": "https://gw.mcpzero.io/v1/ep_self" }
  }
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runServers([]string{
		"--mcp-config", path,
		"--endpoint", "ep_self",
		"--gw-base", "https://gw.mcpzero.io",
	})
	if err == nil || !strings.Contains(err.Error(), "loop check failed") {
		t.Fatalf("expected loop check failure, got %v", err)
	}
}
