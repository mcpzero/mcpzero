package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCursorAddWritesConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")

	if err := cursorAdd([]string{
		"--endpoint", "ep_demo",
		"--api-key", "mz_live_demo",
		"--name", "remote",
		"--config", configPath,
		"--gw-base", "https://gw.example.com",
	}); err != nil {
		t.Fatalf("cursorAdd: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !containsAll(body,
		`"remote"`,
		`"url": "https://gw.example.com/v1/ep_demo"`,
		`"Authorization": "Bearer mz_live_demo"`,
	) {
		t.Fatalf("config body:\n%s", body)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
