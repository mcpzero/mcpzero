---
title: Tunnel
description: Expose a local stdio or HTTP MCP server through the MCPZERO gateway.
---

The tunnel connects your local MCP server to a **Dashboard endpoint** via WebSocket.
It can proxy either a **stdio** MCP server (launched as a subprocess) or an
**HTTP** MCP server (local, or an external server reachable only from your
machine — including ones that require an auth token).

## Prerequisites

- A registered MCPZERO account
- An [endpoint created](/app/endpoints) in the Dashboard (note the endpoint ID, e.g. `ep_dev`)
- CLI logged in (`mcpzero login`) **or** a tunnel token for `--token`

## Start tunnel (stdio)

```bash
mcpzero tunnel start \
  --endpoint ep_abc123 \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp/mcpzero-test"
```

## Start tunnel (HTTP)

Proxy a local HTTP MCP server:

```bash
mcpzero tunnel start \
  --endpoint ep_abc123 \
  --mcp-url http://localhost:6000
```

Proxy an external HTTP MCP server that needs an auth token. Headers are sent on
every upstream request; `${ENV}` references are resolved from the environment:

```bash
mcpzero tunnel start \
  --endpoint ep_abc123 \
  --mcp-url https://api.example.com/mcp \
  --mcp-header "Authorization: Bearer ${UPSTREAM_TOKEN}" \
  --mcp-header "X-Org: acme"
```

Streaming responses (Streamable HTTP / SSE) are relayed end-to-end: the gateway
returns `text/event-stream` to clients that send `Accept: text/event-stream`.

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--endpoint` | Yes | Endpoint ID from Dashboard |
| `--mcp-cmd` | One of | Shell command that starts your MCP server (stdio) |
| `--mcp-url` | One of | URL of an HTTP MCP server to proxy |
| `--mcp-header` | No | HTTP header for `--mcp-url` (repeatable), e.g. `"Authorization: Bearer ${TOKEN}"` |
| `--mcp-transport` | No | HTTP transport: `auto` (default), `streamable-http`, or `sse` (legacy) |
| `--token` | No* | Tunnel token (CI/headless). Omit when logged in. |
| `--gw-base` | No | Gateway URL (default `https://gw.mcpzero.io`) |
| `--detach`, `-d` | No | Run the tunnel in the background as a managed daemon |
| `--force`, `-f` | No | Start even if another tunnel is already running for this endpoint |

Exactly one of `--mcp-cmd` or `--mcp-url` is required.

> **Caution — One tunnel per endpoint:** The gateway keeps a **single active tunnel per endpoint** and evicts the previous
> connection whenever a new one registers. Because daemons auto-reconnect, pointing
> two tunnels at the same endpoint makes them repeatedly evict each other and routes
> requests to whichever upstream currently holds the slot. `tunnel start` therefore
> refuses to start if a running tunnel already exists for the endpoint — stop it
> first (`mcpzero tunnel rm -f <id>`), use a different `--endpoint`, or pass
> `--force` to override.

\* After `mcpzero login`, the CLI sends a refresh token on register instead of `--token`.

### Upstream auth & secrets

The upstream URL and headers (including tokens) are supplied entirely through the
CLI and are **never sent to the gateway**. For background tunnels (`-d`), header
values are stored in the local tunnel state file encrypted with AES-256-GCM
(key at `<config>/mcpzero/state.key`, mode `0600`) so the tunnel can restart
without re-supplying the token.

## Example — filesystem MCP

```bash
mkdir -p /tmp/mcpzero-test

mcpzero tunnel start \
  --endpoint ep_abc123 \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp/mcpzero-test"
```

On macOS, the filesystem server may resolve allowed paths under `/private/tmp/...`.

## Local dev

```bash
# Terminal 1 — gateway
cd saas/gateway && pnpm dev   # :8787

# Terminal 2 — tunnel
mcpzero login --web-base http://localhost:8788 --gw-base http://localhost:8787
mcpzero tunnel start \
  --endpoint ep_dev \
  --gw-base http://localhost:8787 \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp/mcpzero-test"
```

Use seed endpoint `ep_dev` after applying dev migrations (see repo `docs/phase1-e2e.md`).

## What happens under the hood

1. CLI dials `wss://gw.mcpzero.io/tunnel/{endpointId}`
2. Sends `register` with CLI refresh token or tunnel token (plus the upstream transport)
3. Gateway validates ownership and marks endpoint **online**
4. MCP requests from clients are forwarded over WebSocket → CLI → local stdio/HTTP MCP server → streamed back as response chunks

Only **one active tunnel** per endpoint; a new connection replaces the previous one.

## Next

- [Configure Cursor](/docs/cli/cursor/)
- [Troubleshooting](/docs/cli/troubleshooting/)
