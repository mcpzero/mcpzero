package mcpconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverAcrossAgents(t *testing.T) {
	home := t.TempDir()
	work := t.TempDir()

	// Cursor global: filesystem + a shared "git" server.
	writeFile(t, filepath.Join(home, ".cursor", "mcp.json"), `{
      "mcpServers": {
        "filesystem": {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]},
        "git": {"command": "uvx", "args": ["mcp-server-git"]}
      }
    }`)

	// Windsurf (OS-independent path) re-defines the SAME git server -> dedup.
	writeFile(t, filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"), `{
      "mcpServers": {
        "git": {"command": "uvx", "args": ["mcp-server-git"]}
      }
    }`)

	// VS Code project config uses the "servers" key and an HTTP server.
	writeFile(t, filepath.Join(work, ".vscode", "mcp.json"), `{
      "servers": {
        "remote": {"url": "https://remote.example/mcp", "type": "http"}
      }
    }`)

	specs, err := discoverIn(home, work)
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]ServerSpec{}
	for _, s := range specs {
		byName[s.Name] = s
	}

	// git should appear once (deduped) with both sources recorded.
	gitCount := 0
	for _, s := range specs {
		if s.RawName == "git" {
			gitCount++
		}
	}
	if gitCount != 1 {
		t.Fatalf("expected git deduped to 1 entry, got %d (%v)", gitCount, names(specs))
	}

	git := findByRaw(specs, "git")
	if git == nil {
		t.Fatal("git not found")
	}
	if !containsSource(git.Source, "cursor") || !containsSource(git.Source, "windsurf") {
		t.Fatalf("git source should list both cursor and windsurf, got %q", git.Source)
	}

	// HTTP server from VS Code (servers key) is discovered.
	remote := findByRaw(specs, "remote")
	if remote == nil || !remote.IsHTTP() || remote.URL != "https://remote.example/mcp" {
		t.Fatalf("remote http server not discovered correctly: %+v", remote)
	}

	if _, ok := byName["filesystem"]; !ok {
		t.Fatalf("filesystem not discovered: %v", names(specs))
	}
}

func TestDiscoverNothingFound(t *testing.T) {
	home := t.TempDir()
	work := t.TempDir()
	if _, err := discoverIn(home, work); err == nil {
		t.Fatal("expected error when no agent configs exist")
	}
}

func findByRaw(specs []ServerSpec, raw string) *ServerSpec {
	for i := range specs {
		if specs[i].RawName == raw {
			return &specs[i]
		}
	}
	return nil
}

func containsSource(source, agent string) bool {
	for _, part := range splitTrim(source) {
		if part == agent {
			return true
		}
	}
	return false
}

func splitTrim(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, trim(cur))
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, trim(cur))
	return out
}

func trim(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
