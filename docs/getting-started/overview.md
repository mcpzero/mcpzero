---
title: Overview
description: Core concepts — endpoints, tunnels, aggregation, API keys, and the observability ledger.
---

MCPZERO is a **secure MCP aggregation gateway for teams**. It turns local stdio MCP servers into remote HTTP endpoints that AI clients like Cursor can call — with semantic aggregation, progressive discovery, and zero-trust security built in.

## Core concepts

| Concept | What it is |
|---------|------------|
| **Endpoint** | A logical MCP exposure point you create in the [Dashboard](/app/endpoints). Each endpoint has a unique ID (`ep_…`) and public URL. |
| **MCP server** | The tool server process (filesystem, postgres, custom tools) running on your machine via stdio or http. |
| **CLI tunnel** | `mcpzero tunnel start` binds one endpoint to one or more local MCP servers via WebSocket. |
| **Gateway** | Edge worker at `https://gw.mcpzero.io` — authenticates API keys, aggregates servers, forwards JSON-RPC, writes activities. |
| **Semantic aggregation** | When an endpoint multiplexes 2+ servers, the root URL becomes a meta server that routes tool calls intelligently. |
| **Progressive discovery** | Clients discover tools on demand via `meta_search` instead of loading every schema upfront. |
| **API key** | Credential for calling an endpoint (`Authorization: Bearer <mz_live_api_key>`). Generated in [Dashboard → API Keys](/app/api-keys). |
| **Ledger** | Per-call trace of tool name, latency, and status. Payload retention depends on your plan. |

## Typical workflow

1. **Sign in** at [mcpzero.io/app/register](/app/register)
2. **Create an endpoint** in the Dashboard (note the endpoint ID)
3. **Login CLI** — `mcpzero login` (browser flow, no manual token copy)
4. **Start tunnel** — `mcpzero tunnel start --endpoint ep_… --mcp-auto` (reads your mcp.json)
5. **Get an API key** — [Dashboard → API Keys](/app/api-keys)
6. **Configure Cursor** — point at the endpoint root or a specific server path + your API key
7. **Inspect calls** — [Dashboard → Activity](/app/activity)

## Architecture

```
Cursor / AI client
    │  POST /v1/ep_abc123  +  Authorization: Bearer
    ▼
Gateway (gw.mcpzero.io)
    │  semantic aggregation · progressive discovery · auth
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
- [Install the CLI](/docs/cli/install/)
- [Login and start a tunnel](/docs/cli/tunnel/)
- [Configure Cursor](/docs/cli/cursor/)
