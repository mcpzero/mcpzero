package cmd

import (
	"strings"
	"testing"

	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
)

func TestEndpointFromPath(t *testing.T) {
	cases := map[string]string{
		"/v1/ep_abc":              "ep_abc",
		"/v1/ep_abc/files":        "ep_abc",
		"/v1/endpoints/ep_abc":    "ep_abc",
		"/v1/endpoints/ep_abc/fs": "ep_abc",
		"/v1/endpoints":           "",
		"/v2/ep_abc":              "",
		"/":                       "",
		"":                        "",
	}
	for path, want := range cases {
		if got := endpointFromPath(path); got != want {
			t.Errorf("endpointFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestCheckSelfReference(t *testing.T) {
	const gw = "https://gw.mcpzero.io"
	const ep = "ep_self"

	httpSpec := func(name, url string) mcpconfig.ServerSpec {
		return mcpconfig.ServerSpec{Name: name, RawName: name, URL: url}
	}
	stdioSpec := func(name string) mcpconfig.ServerSpec {
		return mcpconfig.ServerSpec{Name: name, RawName: name, Command: "npx server"}
	}

	t.Run("self reference is rejected", func(t *testing.T) {
		specs := []mcpconfig.ServerSpec{
			stdioSpec("local"),
			httpSpec("loopback", "https://gw.mcpzero.io/v1/ep_self"),
		}
		err := checkSelfReference(gw, ep, specs)
		if err == nil {
			t.Fatal("expected an error for a self-referencing endpoint")
		}
		if !strings.Contains(err.Error(), "loopback") {
			t.Errorf("error should name the offending server, got: %v", err)
		}
	})

	t.Run("self reference with server sub-path is rejected", func(t *testing.T) {
		specs := []mcpconfig.ServerSpec{httpSpec("loopback", "https://gw.mcpzero.io/v1/ep_self/files")}
		if err := checkSelfReference(gw, ep, specs); err == nil {
			t.Fatal("expected an error for a self-referencing endpoint sub-path")
		}
	})

	t.Run("different endpoint on same gateway is allowed", func(t *testing.T) {
		specs := []mcpconfig.ServerSpec{httpSpec("chain", "https://gw.mcpzero.io/v1/ep_other")}
		if err := checkSelfReference(gw, ep, specs); err != nil {
			t.Errorf("chaining to a different endpoint should be allowed, got: %v", err)
		}
	})

	t.Run("different host is allowed", func(t *testing.T) {
		specs := []mcpconfig.ServerSpec{httpSpec("external", "https://example.com/v1/ep_self")}
		if err := checkSelfReference(gw, ep, specs); err != nil {
			t.Errorf("a different host should be allowed, got: %v", err)
		}
	})

	t.Run("stdio servers are ignored", func(t *testing.T) {
		specs := []mcpconfig.ServerSpec{stdioSpec("local")}
		if err := checkSelfReference(gw, ep, specs); err != nil {
			t.Errorf("stdio servers should never trip the check, got: %v", err)
		}
	})
}
