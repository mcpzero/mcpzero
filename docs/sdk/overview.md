---
title: SDK overview
description: Tunnel your MCP server from code with the MCPZERO SDKs.
---

The MCPZERO SDKs let you expose your MCP server through the MCPZERO tunnel
**from inside your own code**, in four languages — TypeScript, Python, Go, and
Rust. You get the same result as `mcpzero tunnel start` (a public
Streamable-HTTP endpoint plus dashboard visibility), driven from your
application instead of the CLI.

Each SDK ships **two integration models**:

- **In-process adapter (primary)** — attach the tunnel directly to your existing
  MCP server object; no subprocess, no loopback.
- **Embedded proxy (lower-level)** — open the tunnel and proxy to a local
  stdio/HTTP MCP server (a library version of the CLI tunnel).

In tunnel mode, **visibility is produced by the gateway automatically** — calls
appear in the dashboard exactly as they do for the CLI tunnel
(`visibility_source: "gateway"`), with no extra SDK code.

Configuration is consistent across languages via environment variables:

| Variable | Description |
|----------|-------------|
| `MCPZERO_ENDPOINT_ID` | Endpoint ID from the [Dashboard](/app/endpoints) |
| `MCPZERO_TUNNEL_TOKEN` | Per-endpoint tunnel token from the Dashboard |
| `MCPZERO_GW_BASE` | Optional gateway base (default `https://gw.mcpzero.io`) |

Pick your language:

- [TypeScript tunnel](/docs/sdk/tunnel/typescript/)
- [Python tunnel](/docs/sdk/tunnel/python/)
- [Go tunnel](/docs/sdk/tunnel/go/)
- [Rust tunnel](/docs/sdk/tunnel/rust/)

The wire protocol is specified once in the repo at `sdk/PROTOCOL.md` (tunnel
protocol v2) and shared by every language port.
