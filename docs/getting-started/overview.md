---
title: Overview
description: Core concepts — endpoints, tunnels, API keys, and the observability ledger.
---

MCPZERO turns a **local stdio MCP server** into a **remote HTTP endpoint** that clients like Cursor can call securely.

## Core concepts

| Concept | What it is |
|---------|------------|
| **Endpoint** | A logical MCP exposure point you create in the [Dashboard](/app/endpoints). Each endpoint has a unique ID (`ep_…`) and public URL. |
| **MCP server** | The tool server process (filesystem, postgres, custom tools) running on your machine via stdio. |
| **CLI tunnel** | `mcpzero tunnel start` binds one endpoint to one local MCP server via WebSocket. |
| **Gateway** | Edge worker at `https://gw.mcpzero.io` — authenticates API keys, forwards JSON-RPC, writes the ledger. |
| **API key** | Buyer credential for calling an endpoint (`Authorization: Bearer <mz_live_api_key>`). Generated in [Dashboard → API Keys](/app/api-keys). |
| **Ledger** | Per-call trace of tool name, latency, status, and request/response payloads. |

## Typical workflow

1. **Register** at [mcpzero.io/app/register](/app/register)
2. **Create an endpoint** in the Dashboard (note the endpoint ID)
3. **Login CLI** — `mcpzero login` (browser flow, no manual token copy)
4. **Start tunnel** — `mcpzero tunnel start --endpoint ep_… --mcp-cmd "…"`
5. **Configure Cursor** — remote MCP URL `https://gw.mcpzero.io/v1/ep_abc123/postgres` + your API key
6. **Inspect calls** — [Dashboard → Ledger](/app/ledger)

## Architecture (simplified)

```
Cursor / AI client
    │  POST /v1/ep_abc123/postgres  +  Authorization: Bearer
    ▼
Gateway (gw.mcpzero.io)
    │  WebSocket forward
    ▼
mcpzero CLI  ←→  local MCP server (stdio)
```

## Domains

| Host | Purpose |
|------|---------|
| `mcpzero.io` | Landing, Dashboard, Docs |
| `gw.mcpzero.io` | MCP proxy + tunnel WebSocket |

## Next steps

- [Install the CLI](/docs/cli/install/)
- [Login and start a tunnel](/docs/cli/tunnel/)
- [Configure Cursor](/docs/cli/cursor/)
