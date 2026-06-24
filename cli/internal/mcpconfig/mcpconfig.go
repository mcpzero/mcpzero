// Package mcpconfig parses a standard MCP server configuration file (the
// Cursor/Claude-style `{ "mcpServers": { ... } }` JSON) into a list of server
// specs that the tunnel can proxy. It supports both stdio servers
// (command/args/env) and remote HTTP servers (url/headers/type), normalizes
// each server name into a URL-safe, de-duplicated slug, and offers an
// interactive multi-select helper.
package mcpconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ServerSpec is one resolved MCP server from the config file.
type ServerSpec struct {
	// Name is the normalized, URL-safe, unique server name used in the remote
	// path (/v1/<ep>/<Name>).
	Name string
	// RawName is the original key from the config file (for display).
	RawName string

	// Stdio fields (set when the entry has a "command").
	Command string            // shell command line (command + quoted args)
	WorkDir string            // optional working directory ("cwd")
	Env     map[string]string // environment overrides

	// HTTP fields (set when the entry has a "url").
	URL     string            // remote MCP URL
	Type    string            // raw transport hint: http|streamable-http|sse
	Headers map[string]string // request headers

	// Source describes where this spec was discovered (agent name), set by
	// Discover; empty for an explicitly loaded --mcp-config file.
	Source string
	// SourceFile is the absolute path of the config file it came from.
	SourceFile string
}

// IsHTTP reports whether this spec targets a remote HTTP MCP server.
func (s ServerSpec) IsHTTP() bool { return s.URL != "" }

// signature returns a stable identity for de-duplicating the same server found
// in multiple agent configs (same command, or same URL+transport).
func (s ServerSpec) signature() string {
	if s.IsHTTP() {
		return "http\x00" + s.URL + "\x00" + s.Type
	}
	return "stdio\x00" + s.WorkDir + "\x00" + s.Command
}

// rawEntry mirrors a single value in an mcpServers / servers map.
type rawEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
	Cwd     string            `json:"cwd"`
	WorkDir string            `json:"workdir"`
	URL     string            `json:"url"`
	// serverUrl is an alternate key some agents use for remote servers.
	ServerURL string            `json:"serverUrl"`
	Type      string            `json:"type"`
	Headers   map[string]string `json:"headers"`
}

// defaultKeys are the top-level JSON keys that hold a server map, tried in
// order when a specific key is not supplied. "mcpServers" is the de-facto
// standard (Cursor, Claude, Windsurf, Gemini); "servers" is used by VS Code.
var defaultKeys = []string{"mcpServers", "servers"}

// Load reads and parses a config file path into ordered, normalized specs.
// Server names are sorted alphabetically by their original key for a stable
// selection order.
func Load(path string) ([]ServerSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read --mcp-config %q: %w", path, err)
	}
	return Parse(data)
}

// Parse parses raw config bytes into normalized specs, auto-detecting the
// server-map key (mcpServers or servers).
func Parse(data []byte) ([]ServerSpec, error) {
	entries, err := parseEntries(data, "")
	if err != nil {
		return nil, fmt.Errorf("parse mcp config: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("mcp config has no servers under \"mcpServers\" or \"servers\"")
	}

	specs := make([]ServerSpec, 0, len(entries))
	used := make(map[string]int)
	for _, raw := range sortedEntryNames(entries) {
		spec, err := buildSpec(raw, entries[raw])
		if err != nil {
			return nil, err
		}
		spec.Name = uniqueSlug(slugify(raw), used)
		specs = append(specs, spec)
	}
	return specs, nil
}

// parseEntries extracts the server map from raw config bytes. When key is "",
// the keys in defaultKeys are tried in order. Returns (nil, nil) when no
// matching key holds a non-empty map.
func parseEntries(data []byte, key string) (map[string]rawEntry, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, err
	}
	keys := []string{key}
	if key == "" {
		keys = defaultKeys
	}
	for _, k := range keys {
		raw, ok := top[k]
		if !ok {
			continue
		}
		var m map[string]rawEntry
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		if len(m) > 0 {
			return m, nil
		}
	}
	return nil, nil
}

// buildSpec converts one raw entry into a ServerSpec with Name left unset (the
// caller assigns a unique slug).
func buildSpec(rawName string, entry rawEntry) (ServerSpec, error) {
	spec := ServerSpec{
		RawName: rawName,
		Env:     entry.Env,
		Headers: entry.Headers,
		Type:    entry.Type,
	}
	url := firstNonEmpty(entry.URL, entry.ServerURL)
	switch {
	case entry.Command != "":
		spec.Command = buildCommand(entry.Command, entry.Args)
		spec.WorkDir = firstNonEmpty(entry.Cwd, entry.WorkDir)
	case url != "":
		spec.URL = url
	default:
		return ServerSpec{}, fmt.Errorf("server %q has neither \"command\" nor \"url\"", rawName)
	}
	return spec, nil
}

func sortedEntryNames(entries map[string]rawEntry) []string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SelectServers resolves which specs to start. If preselected names are given,
// it filters to those (matching either the normalized or original name) and
// errors on any unknown name. With a single server it returns it directly.
// Otherwise it prints a numbered prompt and reads a selection ("all" or
// comma/space-separated indices) from in.
func SelectServers(specs []ServerSpec, preselected []string, in io.Reader, out io.Writer) ([]ServerSpec, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("no servers to select")
	}

	if len(preselected) > 0 {
		return filterByNames(specs, preselected)
	}

	if len(specs) == 1 {
		return specs, nil
	}

	fmt.Fprintln(out, "Select which MCP servers to start:")
	for i, s := range specs {
		label := s.RawName
		if s.Name != s.RawName {
			label = fmt.Sprintf("%s (as %q)", s.RawName, s.Name)
		}
		if s.Source != "" {
			label = fmt.Sprintf("%s [%s]", label, s.Source)
		}
		fmt.Fprintf(out, "  [%d] %s -> %s\n", i+1, label, describe(s))
	}
	fmt.Fprint(out, "Enter numbers (e.g. 1,3) or 'all': ")

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read selection: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("no servers selected; pass --mcp-server <name> to choose non-interactively")
	}

	return parseSelection(specs, line)
}

func parseSelection(specs []ServerSpec, line string) ([]ServerSpec, error) {
	if strings.EqualFold(line, "all") {
		return specs, nil
	}

	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t'
	})
	if len(fields) == 0 {
		return nil, fmt.Errorf("no servers selected")
	}

	seen := make(map[int]bool)
	var chosen []ServerSpec
	for _, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 1 || n > len(specs) {
			return nil, fmt.Errorf("invalid selection %q (choose 1-%d or 'all')", f, len(specs))
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		chosen = append(chosen, specs[n-1])
	}
	if len(chosen) == 0 {
		return nil, fmt.Errorf("no servers selected")
	}
	return chosen, nil
}

func filterByNames(specs []ServerSpec, names []string) ([]ServerSpec, error) {
	byName := make(map[string]ServerSpec, len(specs)*2)
	for _, s := range specs {
		byName[s.Name] = s
		byName[s.RawName] = s
	}

	seen := make(map[string]bool)
	var chosen []ServerSpec
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		s, ok := byName[name]
		if !ok {
			// Also try matching the slugified form of the requested name.
			s, ok = byName[slugify(name)]
		}
		if !ok {
			return nil, fmt.Errorf("--mcp-server %q not found in config", raw)
		}
		if seen[s.Name] {
			continue
		}
		seen[s.Name] = true
		chosen = append(chosen, s)
	}
	if len(chosen) == 0 {
		return nil, fmt.Errorf("no matching servers for --mcp-server")
	}
	return chosen, nil
}

func describe(s ServerSpec) string {
	if s.IsHTTP() {
		return s.URL
	}
	return s.Command
}

// buildCommand joins a command and its args into a single shell command line,
// quoting each token so the stdio bridge (`sh -c <line>`) runs it verbatim.
func buildCommand(command string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(command))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// shellQuote wraps s in single quotes, escaping embedded single quotes, so it
// is treated as one literal token by sh.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`&|;<>()*?[]{}#~!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// slugify converts a server name into a URL-safe slug containing only
// [A-Za-z0-9_-], collapsing runs of other characters into a single '-'.
func slugify(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "server"
	}
	return slug
}

// uniqueSlug ensures slug is unique within used, appending -2, -3, … on
// collision. It records the chosen slug in used.
func uniqueSlug(slug string, used map[string]int) string {
	if _, ok := used[slug]; !ok {
		used[slug] = 1
		return slug
	}
	for n := used[slug] + 1; ; n++ {
		candidate := fmt.Sprintf("%s-%d", slug, n)
		if _, ok := used[candidate]; !ok {
			used[slug] = n
			used[candidate] = 1
			return candidate
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
