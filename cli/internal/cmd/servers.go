package cmd

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/mcpzero/mcpzero/cli/internal/daemon"
	"github.com/mcpzero/mcpzero/cli/internal/mcpconfig"
	tunnelpkg "github.com/mcpzero/mcpzero/cli/internal/tunnel"
	"github.com/mcpzero/mcpzero/cli/internal/upstream"
)

// onStartFunc returns the onStart callback for a given server name (used to
// record the stdio process-group pid). It may return nil.
type onStartFunc func(name string) func(pid int)

// checkSelfReference rejects a config whose HTTP server points back at this
// tunnel's own gateway endpoint. Such an entry would make the gateway forward a
// request to this tunnel, which forwards it right back to the same endpoint —
// an infinite request loop. We only flag a direct self-reference (same gateway
// host + same endpoint id); pointing at a *different* endpoint is a legitimate
// chained proxy. The gateway enforces a runtime backstop (X-MCP-Loop) for
// cross-endpoint and multi-hop cycles this static check cannot see.
func checkSelfReference(gwBase, endpointID string, specs []mcpconfig.ServerSpec) error {
	gwURL, err := url.Parse(gwBase)
	if err != nil || gwURL.Hostname() == "" {
		// Best-effort: if we can't resolve the gateway host, skip the check.
		return nil
	}
	for _, spec := range specs {
		if !spec.IsHTTP() {
			continue
		}
		u, err := url.Parse(spec.URL)
		if err != nil {
			continue
		}
		if !strings.EqualFold(u.Hostname(), gwURL.Hostname()) {
			continue
		}
		if endpointFromPath(u.Path) != endpointID {
			continue
		}
		return fmt.Errorf(
			"server %q (%s) points back at this tunnel's own endpoint %s; "+
				"the gateway would forward requests to this tunnel and straight back "+
				"to the same endpoint, an infinite loop. Remove this entry from the "+
				"config, or point it at a different endpoint.",
			spec.RawName, spec.URL, endpointID,
		)
	}
	return nil
}

// endpointFromPath extracts the endpoint id from a gateway MCP URL path,
// supporting both the canonical /v1/<id>[/<server>] and the legacy
// /v1/endpoints/<id>[/<server>] forms. Returns "" when the path is not a
// gateway MCP path.
func endpointFromPath(p string) string {
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

// namedUpstreamsFromSpecs builds tunnel upstreams from selected config specs.
func namedUpstreamsFromSpecs(
	specs []mcpconfig.ServerSpec,
	onStartFor onStartFunc,
) ([]tunnelpkg.NamedUpstream, error) {
	out := make([]tunnelpkg.NamedUpstream, 0, len(specs))
	for _, spec := range specs {
		var (
			up  upstream.Upstream
			err error
		)
		var onStart func(pid int)
		if onStartFor != nil {
			onStart = onStartFor(spec.Name)
		}

		if spec.IsHTTP() {
			headers, herr := configHeaders(spec.Headers)
			if herr != nil {
				return nil, fmt.Errorf("server %q: %w", spec.Name, herr)
			}
			up, err = upstream.NewHTTP(spec.URL, headers, transportFromConfigType(spec.Type))
		} else {
			up = upstream.NewStdio(spec.Command, spec.WorkDir, envMapToStrings(spec.Env), onStart)
		}
		if err != nil {
			return nil, fmt.Errorf("server %q: %w", spec.Name, err)
		}
		out = append(out, tunnelpkg.NamedUpstream{Name: spec.Name, Upstream: up})
	}
	return out, nil
}

// daemonServersFromSpecs converts config specs into the daemon spawn form,
// resolving headers (with ${ENV} interpolation) so values can be encrypted.
func daemonServersFromSpecs(specs []mcpconfig.ServerSpec) ([]daemon.ServerSpec, error) {
	out := make([]daemon.ServerSpec, 0, len(specs))
	for _, spec := range specs {
		ds := daemon.ServerSpec{
			Name:       spec.Name,
			MCPWorkDir: spec.WorkDir,
			Env:        envMapToHeaders(spec.Env),
		}
		if spec.IsHTTP() {
			headers, err := configHeaders(spec.Headers)
			if err != nil {
				return nil, fmt.Errorf("server %q: %w", spec.Name, err)
			}
			ds.MCPURL = spec.URL
			ds.MCPTransport = transportFromConfigType(spec.Type)
			ds.MCPHeaders = headers
		} else {
			ds.MCPCmd = spec.Command
		}
		out = append(out, ds)
	}
	return out, nil
}

// namedUpstreamsFromState rebuilds tunnel upstreams from persisted server state.
func namedUpstreamsFromState(
	servers []daemon.ServerState,
	onStartFor onStartFunc,
) ([]tunnelpkg.NamedUpstream, error) {
	out := make([]tunnelpkg.NamedUpstream, 0, len(servers))
	for i := range servers {
		sv := servers[i]
		var (
			up  upstream.Upstream
			err error
		)
		var onStart func(pid int)
		if onStartFor != nil {
			onStart = onStartFor(sv.Name)
		}

		if sv.MCPURL != "" {
			headers, herr := sv.Headers()
			if herr != nil {
				return nil, fmt.Errorf("server %q: %w", sv.Name, herr)
			}
			up, err = upstream.NewHTTP(sv.MCPURL, headers, sv.MCPTransport)
		} else {
			env, eerr := sv.Env()
			if eerr != nil {
				return nil, fmt.Errorf("server %q: %w", sv.Name, eerr)
			}
			up = upstream.NewStdio(sv.MCPCmd, sv.MCPWorkDir, headersToStrings(env), onStart)
		}
		if err != nil {
			return nil, fmt.Errorf("server %q: %w", sv.Name, err)
		}
		out = append(out, tunnelpkg.NamedUpstream{Name: sv.Name, Upstream: up})
	}
	return out, nil
}

// configHeaders converts a config header map into upstream headers, resolving
// ${ENV} references in values (via upstream.ParseHeader).
func configHeaders(m map[string]string) ([]upstream.Header, error) {
	if len(m) == 0 {
		return nil, nil
	}
	names := sortedKeys(m)
	out := make([]upstream.Header, 0, len(names))
	for _, name := range names {
		h, err := upstream.ParseHeader(fmt.Sprintf("%s: %s", name, m[name]))
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

// envMapToHeaders converts an env map to name/value pairs (sorted for stable
// ordering) for encrypted persistence.
func envMapToHeaders(m map[string]string) []upstream.Header {
	if len(m) == 0 {
		return nil
	}
	names := sortedKeys(m)
	out := make([]upstream.Header, 0, len(names))
	for _, name := range names {
		out = append(out, upstream.Header{Name: name, Value: m[name]})
	}
	return out
}

// envMapToStrings converts an env map to "KEY=VALUE" entries (sorted).
func envMapToStrings(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	names := sortedKeys(m)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, name+"="+m[name])
	}
	return out
}

// headersToStrings converts name/value pairs to "KEY=VALUE" entries.
func headersToStrings(hs []upstream.Header) []string {
	if len(hs) == 0 {
		return nil
	}
	out := make([]string, 0, len(hs))
	for _, h := range hs {
		out = append(out, h.Name+"="+h.Value)
	}
	return out
}

func transportFromConfigType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "sse":
		return upstream.TransportSSE
	case "http", "streamable-http", "streamable_http", "streamablehttp":
		return upstream.TransportStreamable
	default:
		return upstream.TransportAuto
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
