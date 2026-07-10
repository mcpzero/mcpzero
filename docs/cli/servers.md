---
title: Servers
description: List and validate MCP config before starting a tunnel (loop dry-run).
---

`mcpzero servers` parses an MCP config the same way `tunnel start` does, then
prints every server it would register. Pass `--endpoint` to dry-run the
self-loop check without opening a WebSocket.

## Usage

```bash
# From a config file
mcpzero servers --mcp-config ./mcp.json

# Auto-discover Cursor / Claude Desktop / Codex configs
mcpzero servers --mcp-auto

# Also fail if any HTTP server points back at this endpoint
mcpzero servers --mcp-config ./mcp.json --endpoint ep_abc123
```

## Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--mcp-config` | One of | Path to MCP config JSON (`mcpServers` / `servers`) |
| `--mcp-auto` | One of | Discover servers from installed agent configs |
| `--endpoint` | No | Endpoint ID for self-loop validation |
| `--gw-base` | No | Gateway base for loop check (default: credentials or production) |

## Loop check

When `--endpoint` is set, the CLI rejects configs that would tunnel an HTTP
server whose URL is this endpoint (or cluster) on the gateway — the same
guard used by `tunnel start`. Without `--endpoint`, listing still works and
prints `loop check: skipped`.

## Example

```text
source: ./mcp.json
servers: 2

NAME     TRANSPORT  DETAIL
fs       stdio      npx
remote   http       https://example.com/mcp

loop check: ok (no server points at https://gw.mcpzero.io/ep_abc123)
```

## See also

- [Tunnel](/docs/cli/tunnel/) — start a multiplexed tunnel with the same flags
- [Doctor](/docs/cli/doctor/) — broader health checks
- [Init](/docs/cli/init/) — first-time onboarding
