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
- CLI logged in (`mcpzero login`) **or** a management key for `--mgmt-key`

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
| `--mcp-config` | One of | Path to an MCP config file (`mcpServers` JSON); multiplex selected servers over one tunnel |
| `--mcp-auto` | One of | Auto-discover MCP servers from Cursor / Claude Desktop / Codex configs and choose interactively |
| `--mcp-server` | No | Server name from `--mcp-config` / `--mcp-auto` to start (repeatable; skips the interactive prompt) |
| `--mcp-workdir` | No | Working directory for `--mcp-cmd` |
| `--mcp-header` | No | HTTP header for `--mcp-url` (repeatable), e.g. `"Authorization: Bearer ${TOKEN}"` |
| `--mcp-transport` | No | HTTP transport: `auto` (default), `streamable-http`, or `sse` (legacy) |
| `--mgmt-key` | No* | Management key for CI/headless (or set `MCPZERO_MGMT_KEY`). Omit when logged in. |
| `--gw-base` | No | Gateway URL (default `https://gw.mcpzero.io`) |
| `--detach`, `-d` | No | Run the tunnel in the background as a managed daemon |
| `--force`, `-f` | No | Start even if another tunnel is already running for this endpoint |

Exactly one of `--mcp-cmd`, `--mcp-url`, `--mcp-config`, or `--mcp-auto` is required. `--mcp-config` and `--mcp-auto` are mutually exclusive and cannot be combined with `--mcp-cmd` or `--mcp-url`.

> **Caution — One tunnel per endpoint:** The gateway keeps a **single active tunnel per endpoint** and evicts the previous
> connection whenever a new one registers. Because daemons auto-reconnect, pointing
> two tunnels at the same endpoint makes them repeatedly evict each other and routes
> requests to whichever upstream currently holds the slot. `tunnel start` therefore
> refuses to start if a running tunnel already exists for the endpoint — stop it
> first (`mcpzero tunnel rm -f <id>`), use a different `--endpoint`, or pass
> `--force` to override.

\* After `mcpzero login`, the CLI sends a refresh token on register instead of `--mgmt-key`.

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

## Multiplexed tunnel (`--mcp-config` / `--mcp-auto`)

Both flags start a **multiplexed tunnel** — multiple MCP servers over one WebSocket connection. Each selected server gets its own public path (`/v1/<endpoint>/<server>`). When one or more servers are registered, the endpoint **root** is also a meta server for semantic aggregation and progressive discovery. See [Semantic aggregation](/docs/gateway/semantic-aggregation/).

| Flag | How servers are found |
|------|------------------------|
| `--mcp-config <file>` | Read `mcpServers` from a JSON file (same format as Cursor / Claude Desktop) |
| `--mcp-auto` | Scan installed agent configs (Cursor, Claude Desktop, Codex, etc.) in your home directory and current project |

### Interactive server selection

When more than one server is available, the CLI **prompts you to choose** which to start — it does not silently tunnel everything.

- Enter numbers: `1,3` or `3,7,8`
- Type `all` or press **Enter** to select every listed server
- Pass `--mcp-server <name>` (repeatable) to skip the prompt

### From a config file (`--mcp-config`)

```bash
mcpzero tunnel start --endpoint ep_abc123 --mcp-config ./mcp.json
```

Use the [sample `mcp.json`](https://github.com/mcpzero/mcpzero/blob/main/examples/quickstart/mcp.json) or your own file. If the file defines multiple servers, you get the selection prompt. Non-interactive example:

```bash
mcpzero tunnel start --endpoint ep_abc123 \
  --mcp-config ./mcp.json \
  --mcp-server local-fs-tmp \
  --mcp-server playwright-headless
```

### Auto-discover (`--mcp-auto`)

```bash
mcpzero tunnel start --endpoint ep_abc123 --mcp-auto
```

The CLI prints how many servers it found in agent configs, lists them with source tags (`[cursor]`, `[project]`, …), and asks which to tunnel.

### Example CLI output (`--mcp-auto`)

```
discovered 8 MCP server(s) from agent configs
Select which MCP servers to start:
  [1] docker-fs [cursor] -> https://gw.mcpzero.io/v1/ep_34e06ce26b
  [2] docker-fs-3 [cursor] -> https://gw.mcpzero.io/v1/ep_0476da16c9/local-fs-tmp
  [3] local-fs-tmp [cursor] -> npx -y @modelcontextprotocol/server-filesystem /tmp
  [4] local-mcpzero-bundle [cursor] -> http://127.0.0.1:7901/v1/epc_2b94b1088a  ← cluster URL (client target, not a tunnel)
  [5] mcpzero-bundle [cursor] -> https://gw.mcpzero.io/v1/ep_0476da16c9
  [6] playwright-headless [cursor] -> https://gw.mcpzero.io/v1/ep_0476da16c9/playwright-headless
  [7] local-fs-tmp (as "local-fs-tmp-2") [project] -> npx -y @modelcontextprotocol/server-filesystem /path/to/tmp
  [8] playwright-headless (as "playwright-headless-2") [project] -> npx @playwright/mcp@latest
Enter numbers (e.g. 1,3), 'all', or press Enter for all: 3,7,8
selected 3 server(s): local-fs-tmp, local-fs-tmp-2, playwright-headless-2
remote MCP URLs:
  local-fs-tmp          https://gw.mcpzero.io/v1/ep_0476da16c9/local-fs-tmp
  local-fs-tmp-2        https://gw.mcpzero.io/v1/ep_0476da16c9/local-fs-tmp-2
  playwright-headless-2 https://gw.mcpzero.io/v1/ep_0476da16c9/playwright-headless-2
connecting to https://gw.mcpzero.io/tunnel/ep_0476da16c9 …
[2026-07-04 22:10:51] tunnel registered for endpoint ep_0476da16c9
```

Each line shows the server name, where it was discovered, and the local command or existing remote URL. Lines pointing at `epc_*` are **cluster roots** already configured in Cursor — select them only if you intend to proxy that remote URL; normally you tunnel local stdio servers. See [Endpoint clusters](/docs/gateway/endpoint-clusters/). After you connect, point Cursor at the **endpoint root** for meta server features:

`https://gw.mcpzero.io/v1/ep_0476da16c9`

## What happens under the hood

1. CLI dials `wss://gw.mcpzero.io/tunnel/{endpointId}`
2. Sends `register` with CLI refresh token or management key (plus the upstream transport)
3. Gateway validates ownership and marks endpoint **online**
4. MCP requests from clients are forwarded over WebSocket → CLI → local stdio/HTTP MCP server → streamed back as response chunks

Only **one active tunnel** per endpoint; a new connection replaces the previous one.

## Next

- [Configure Cursor](/docs/cli/cursor/)
- [Troubleshooting](/docs/cli/troubleshooting/)
