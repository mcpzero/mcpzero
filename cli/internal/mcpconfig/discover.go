package mcpconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// candidate is one known agent config file location and the JSON key holding
// its MCP server map.
type candidate struct {
	agent string // human-readable agent name (shown in the picker)
	path  string // absolute path to the config file
	key   string // top-level key with the server map ("" = auto-detect)
}

// Catalog of major AI agents and where they keep MCP server configs.
//
// Format note: nearly all agents use `{ "mcpServers": { name: {command,args,
// env} | {url,headers,type} } }`. VS Code uses the same entry shape under a
// `servers` key. Agents with a fundamentally different per-server schema (e.g.
// Zed's `context_servers`) are intentionally omitted.
func buildCatalog(home, workDir string) []candidate {
	var c []candidate

	// ---- Home-scoped (global) configs ----
	if home != "" {
		c = append(c,
			// Cursor (global)
			candidate{"cursor", filepath.Join(home, ".cursor", "mcp.json"), "mcpServers"},
			// Claude Code (global)
			candidate{"claude-code", filepath.Join(home, ".claude.json"), "mcpServers"},
			// Windsurf (Codeium)
			candidate{"windsurf", filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"), "mcpServers"},
			// Gemini CLI
			candidate{"gemini-cli", filepath.Join(home, ".gemini", "settings.json"), "mcpServers"},
			// Continue
			candidate{"continue", filepath.Join(home, ".continue", "config.json"), "mcpServers"},
		)
		c = append(c, claudeDesktopCandidate(home))
	}

	// ---- Project-scoped (current directory) configs ----
	if workDir != "" {
		c = append(c,
			// Cursor (project)
			candidate{"cursor (project)", filepath.Join(workDir, ".cursor", "mcp.json"), "mcpServers"},
			// VS Code (project) — uses the "servers" key
			candidate{"vscode (project)", filepath.Join(workDir, ".vscode", "mcp.json"), "servers"},
			// Claude Code (project)
			candidate{"claude-code (project)", filepath.Join(workDir, ".mcp.json"), "mcpServers"},
			// Gemini CLI (project)
			candidate{"gemini-cli (project)", filepath.Join(workDir, ".gemini", "settings.json"), "mcpServers"},
			// Continue (project)
			candidate{"continue (project)", filepath.Join(workDir, ".continue", "config.json"), "mcpServers"},
			// Generic project-level config
			candidate{"project", filepath.Join(workDir, "mcp.json"), ""},
		)
	}

	return c
}

// claudeDesktopCandidate returns the OS-specific Claude Desktop config path.
func claudeDesktopCandidate(home string) candidate {
	const key = "mcpServers"
	switch runtime.GOOS {
	case "darwin":
		return candidate{"claude-desktop", filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), key}
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return candidate{"claude-desktop", filepath.Join(base, "Claude", "claude_desktop_config.json"), key}
	default:
		return candidate{"claude-desktop", filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), key}
	}
}

// Discover scans the known agent config locations under the user's home
// directory and the given working directory, returning a de-duplicated, name-
// unique list of MCP servers across all of them. Each spec is tagged with the
// agent(s) it was found in. Missing or malformed files are skipped.
func Discover(workDir string) ([]ServerSpec, error) {
	home, _ := os.UserHomeDir()
	return discoverIn(home, workDir)
}

// discoverIn is Discover with an explicit home directory (for testing).
func discoverIn(home, workDir string) ([]ServerSpec, error) {
	candidates := buildCatalog(home, workDir)

	used := make(map[string]int) // slug -> dedup counter (global uniqueness)
	seen := make(map[string]int) // signature -> index into result
	var result []ServerSpec

	for _, cand := range candidates {
		data, err := os.ReadFile(cand.path)
		if err != nil {
			continue // missing file (or unreadable): skip
		}
		entries, err := parseEntries(data, cand.key)
		if err != nil || len(entries) == 0 {
			continue // malformed or no servers: skip this file
		}

		for _, raw := range sortedEntryNames(entries) {
			spec, err := buildSpec(raw, entries[raw])
			if err != nil {
				continue // skip individual malformed entries
			}
			sig := spec.signature()
			if idx, ok := seen[sig]; ok {
				// Same server in another agent's config: record the extra source.
				result[idx].Source = mergeSource(result[idx].Source, cand.agent)
				continue
			}
			spec.Name = uniqueSlug(slugify(raw), used)
			spec.Source = cand.agent
			spec.SourceFile = cand.path
			seen[sig] = len(result)
			result = append(result, spec)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf(
			"no MCP servers found in known agent configs under %s or %s",
			displayHome(home), workDir,
		)
	}
	return result, nil
}

// mergeSource appends an additional source agent to a comma-separated list,
// avoiding duplicates.
func mergeSource(existing, add string) string {
	if existing == "" {
		return add
	}
	for _, part := range strings.Split(existing, ",") {
		if strings.TrimSpace(part) == add {
			return existing
		}
	}
	return existing + ", " + add
}

func displayHome(home string) string {
	if home == "" {
		return "~"
	}
	return home
}
