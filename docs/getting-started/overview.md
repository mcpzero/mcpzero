---
title: Overview
description: Core concepts — endpoints, tunnels, aggregation, API keys, and the observability ledger.
---

MCPZERO is a **secure MCP aggregation gateway**. It turns local MCP servers into remote HTTP endpoints that AI clients like Cursor can call — with semantic aggregation, progressive discovery, and zero-trust security built in.

**In one sentence:** MCPZERO publishes, aggregates, and secures local MCP servers behind one zero-trust gateway URL so agents can discover tools on demand instead of loading every schema upfront.

Compare MCPZERO with [direct MCP](/docs/compare/vs-direct-mcp/), [tunnel-only tools](/docs/compare/vs-tunnel-only/), or the full [MCP gateway comparison hub](/docs/compare/) (including [ngrok vs Microsoft vs MCPZERO](/docs/compare/mcp-gateways-2026/)). See also [use cases](/docs/use-cases/multi-server-cursor/) and the [FAQ](/docs/faq/).

## Core concepts

| Concept | What it is |
|---------|------------|
| **Endpoint** | A logical MCP exposure point you create in the [Dashboard](/app/endpoints). Each endpoint has a unique ID (`ep_…`) and public URL. |
| **MCP server** | The tool server process (filesystem, postgres, custom tools) running on your machine via stdio or http. |
| **CLI tunnel** | `mcpzero tunnel start` binds one endpoint to one or more local MCP servers via WebSocket. |
| **Gateway** | Edge worker at `https://gw.mcpzero.io` — authenticates API keys, aggregates servers, forwards JSON-RPC, writes activities. |
| **Meta server** | The endpoint **root** URL (`/v1/ep_abc123`) — exposes `meta_search`, `meta_call_tool`, and server profiles. Enabled by default whenever a server is registered (including a single server). |
| **Semantic aggregation** | Backend server(s) behind one endpoint; the root URL is the meta server (including a single server with many tools). |
| **Progressive discovery** | Clients discover tools on demand via `meta_search` at the endpoint root instead of loading every schema upfront. |
| **API key** | Credential for calling an endpoint (`Authorization: Bearer <mz_live_api_key>`). Generated in [Dashboard → API Keys](/app/api-keys). |
| **Endpoint cluster** | A virtual meta server (`epc_*`) that aggregates **multiple endpoints** for cross-endpoint semantic search and progressive discovery. Team+ only — see [Endpoint clusters](/docs/gateway/endpoint-clusters/). |
| **CLI login** | `mcpzero login` stores a refresh token on your machine so the CLI can start tunnels — **not** used as the Cursor HTTP credential. |
| **Ledger** | Per-call trace of tool name, latency, and status. Payload retention depends on your plan. |

## One-minute setup

**Recommended:** use the interactive CLI wizard after install:

```bash
mcpzero login
mcpzero init
```

`init` walks you through endpoint, API key, Cursor `mcp.json`, and an optional
tunnel. See [Init (onboarding)](/docs/cli/init/).

### Manual setup

Get from zero to a working meta server test without the wizard:

1. **Sign in** at [mcpzero.io/app/register](/app/register)
2. **Create an endpoint** in the Dashboard (note the endpoint ID, e.g. `ep_abc123`)
3. **Install & login CLI** — see [Install](/docs/cli/install/) then `mcpzero login`
4. **Start tunnel** — copy the [sample `mcp.json`](https://github.com/mcpzero/mcpzero/blob/main/examples/quickstart/mcp.json) (or save it as `./mcp.json` next to your project), then run:

   ```bash
   mcpzero tunnel start --endpoint ep_abc123 --mcp-config ./mcp.json
   ```

   Example config (two stdio servers — filesystem + Playwright):

   ```json
   {
     "mcpServers": {
       "playwright-headless": {
         "command": "npx",
         "args": ["@playwright/mcp@latest"]
       },
       "local-fs-tmp": {
         "command": "npx",
         "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
       }
     }
   }
   ```
5. **Create an API key** — [Dashboard → API Keys](/app/api-keys) (`mz_live_…`)
6. **Configure Cursor** — point at the **endpoint root** (recommended):

   `https://gw.mcpzero.io/v1/ep_abc123`

7. **Verify the meta server** with curl:

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

You should see `meta_search` and `meta_call_tool` — not the full backend tool list.

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":2,"method":"meta_search","params":{"intent":"list files"}}'
```

Free includes semantic aggregation and progressive discovery on a single endpoint when you use the root URL. See [Semantic aggregation](/docs/gateway/semantic-aggregation/) and [Progressive discovery](/docs/gateway/progressive-discovery/) for root vs. server URL details.

## Typical workflow

1. **Install CLI** — [Install](/docs/cli/install/)
2. **Login** — `mcpzero login`
3. **Onboard** — `mcpzero init` (endpoint, API key, Cursor, tunnel)
4. **Use Cursor** — tools call through `https://gw.mcpzero.io/v1/ep_…`
5. **Inspect calls** — [Dashboard → Activity](/app/activity)

Or configure step-by-step in the Dashboard:

1. **Sign in** at [mcpzero.io/app/register](/app/register)
2. **Create an endpoint** in the Dashboard (note the endpoint ID)
3. **Login CLI** — `mcpzero login`
4. **Start tunnel** — `mcpzero tunnel start --endpoint ep_… --mcp-auto`
5. **Get an API key** — [Dashboard → API Keys](/app/api-keys)
6. **Configure Cursor** — [Cursor setup](/docs/cli/cursor/) or `mcpzero cursor add`
7. **Inspect calls** — [Dashboard → Activity](/app/activity)

## Architecture

```
Cursor / AI client
    │  POST /v1/ep_abc123  +  Authorization: Bearer
    ▼
Gateway (gw.mcpzero.io)
    │  meta server · semantic aggregation · progressive discovery · auth
    │  WebSocket forward
    ▼
mcpzero CLI  ←→  local MCP servers (stdio)
```

## Domains

| Host | Purpose |
|------|---------|
| `mcpzero.io` | Landing, Dashboard, Docs |
| `gw.mcpzero.io` | MCP gateway + tunnel WebSocket |

## Next steps

- [Semantic aggregation](/docs/gateway/semantic-aggregation/)
- [Progressive discovery](/docs/gateway/progressive-discovery/)
- [Endpoint clusters](/docs/gateway/endpoint-clusters/)
- [Init (onboarding)](/docs/cli/init/)
- [Install the CLI](/docs/cli/install/)
- [Login and start a tunnel](/docs/cli/tunnel/)
- [Configure Cursor](/docs/cli/cursor/)
