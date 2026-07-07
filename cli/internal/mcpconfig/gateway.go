package mcpconfig

import (
	"net/url"
	"strings"
)

// EndpointIDFromGatewayURL extracts the endpoint id from a MCPZERO gateway MCP
// URL path (/v1/<id>[/<server>] or /v1/endpoints/<id>[/<server>]). Returns ""
// when the URL is not a gateway MCP route.
func EndpointIDFromGatewayURL(rawURL string) string {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return endpointIDFromPath(u.Path)
}

func endpointIDFromPath(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	if len(parts) < 2 || parts[0] != "v1" {
		return ""
	}
	if parts[1] == "endpoints" {
		if len(parts) < 3 {
			return ""
		}
		return parts[2]
	}
	return parts[1]
}

// IsPublishedGatewayMCPURL reports whether rawURL points at an already-published
// MCPZERO gateway endpoint. These entries are Cursor/client connection targets
// (written by mcpzero init/cursor add) and must not be offered as --mcp-auto
// tunnel sources.
func IsPublishedGatewayMCPURL(rawURL string) bool {
	id := EndpointIDFromGatewayURL(rawURL)
	return strings.HasPrefix(id, "ep_") || strings.HasPrefix(id, "epc_")
}
