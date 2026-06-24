package mcpconfig

import (
	"strings"
	"testing"
)

func TestParseStdioAndHTTP(t *testing.T) {
	data := []byte(`{
      "mcpServers": {
        "aws-mcp": {
          "command": "uvx",
          "args": ["mcp-proxy-for-aws@latest", "https://aws-mcp.example/mcp", "--metadata", "AWS_REGION=us-west-2"],
          "env": {"AWS_PROFILE": "dev"}
        },
        "remote": {
          "url": "https://remote.example/mcp",
          "type": "streamable-http",
          "headers": {"Authorization": "Bearer xyz"}
        }
      }
    }`)

	specs, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}

	byName := map[string]ServerSpec{}
	for _, s := range specs {
		byName[s.Name] = s
	}

	aws, ok := byName["aws-mcp"]
	if !ok {
		t.Fatalf("missing aws-mcp; got %v", names(specs))
	}
	if aws.IsHTTP() {
		t.Fatalf("aws-mcp should be stdio")
	}
	if !strings.Contains(aws.Command, "uvx") || !strings.Contains(aws.Command, "mcp-proxy-for-aws@latest") {
		t.Fatalf("unexpected command: %q", aws.Command)
	}
	// args with special chars should be quoted into one token boundary
	if !strings.Contains(aws.Command, "'AWS_REGION=us-west-2'") && !strings.Contains(aws.Command, "AWS_REGION=us-west-2") {
		t.Fatalf("metadata arg not present: %q", aws.Command)
	}
	if aws.Env["AWS_PROFILE"] != "dev" {
		t.Fatalf("env not parsed: %v", aws.Env)
	}

	remote, ok := byName["remote"]
	if !ok {
		t.Fatalf("missing remote")
	}
	if !remote.IsHTTP() {
		t.Fatalf("remote should be http")
	}
	if remote.URL != "https://remote.example/mcp" {
		t.Fatalf("unexpected url: %q", remote.URL)
	}
	if remote.Headers["Authorization"] != "Bearer xyz" {
		t.Fatalf("headers not parsed: %v", remote.Headers)
	}
}

func TestSlugifyAndDedup(t *testing.T) {
	data := []byte(`{
      "mcpServers": {
        "My Server!": {"command": "a"},
        "My/Server": {"command": "b"},
        "my-server": {"command": "c"}
      }
    }`)
	specs, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, s := range specs {
		if !isURLSafe(s.Name) {
			t.Errorf("name %q is not URL-safe", s.Name)
		}
		if seen[s.Name] {
			t.Errorf("duplicate normalized name %q", s.Name)
		}
		seen[s.Name] = true
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 unique names, got %d: %v", len(seen), seen)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte(`{}`)); err == nil {
		t.Error("expected error for empty config")
	}
	if _, err := Parse([]byte(`{"mcpServers":{"x":{}}}`)); err == nil {
		t.Error("expected error for server without command or url")
	}
}

func TestSelectServers(t *testing.T) {
	specs := []ServerSpec{
		{Name: "a", RawName: "a", Command: "a"},
		{Name: "b", RawName: "b", Command: "b"},
		{Name: "c", RawName: "c", Command: "c"},
	}

	// Preselected by name.
	got, err := SelectServers(specs, []string{"a", "c"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "c" {
		t.Fatalf("unexpected preselected result: %v", names(got))
	}

	// Unknown preselected name errors.
	if _, err := SelectServers(specs, []string{"nope"}, nil, nil); err == nil {
		t.Error("expected error for unknown --mcp-server name")
	}

	// Interactive: 'all'.
	got, err = SelectServers(specs, nil, strings.NewReader("all\n"), &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected all 3, got %d", len(got))
	}

	// Interactive: indices.
	got, err = SelectServers(specs, nil, strings.NewReader("1,3\n"), &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "c" {
		t.Fatalf("unexpected index selection: %v", names(got))
	}

	// Single server returns directly without prompt.
	got, err = SelectServers(specs[:1], nil, nil, nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("single-server selection failed: %v %v", got, err)
	}
}

func names(specs []ServerSpec) []string {
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}

func isURLSafe(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
