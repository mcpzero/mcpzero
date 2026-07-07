package mcpconfig

import "testing"

func TestEndpointIDFromGatewayURL(t *testing.T) {
	cases := map[string]string{
		"https://gw.mcpzero.io/v1/ep_abc":              "ep_abc",
		"https://gw-dev.mcpzero.io/v1/ep_abc/files":    "ep_abc",
		"https://gw.mcpzero.io/v1/endpoints/ep_abc":    "ep_abc",
		"https://gw.mcpzero.io/v1/epc_cluster":         "epc_cluster",
		"https://remote.example/mcp":                   "",
		"http://localhost:6000/mcp":                    "",
	}
	for raw, want := range cases {
		if got := EndpointIDFromGatewayURL(raw); got != want {
			t.Fatalf("%q: got %q want %q", raw, got, want)
		}
	}
}

func TestIsPublishedGatewayMCPURL(t *testing.T) {
	if !IsPublishedGatewayMCPURL("https://gw.mcpzero.io/v1/ep_test") {
		t.Fatal("expected gateway endpoint URL")
	}
	if IsPublishedGatewayMCPURL("https://api.example.com/mcp") {
		t.Fatal("external HTTP MCP should not be filtered")
	}
}
