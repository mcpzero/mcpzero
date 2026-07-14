---
title: MCPZERO vs direct MCP
description: Compare MCPZERO secure aggregation gateway with connecting MCP servers directly from the AI client.
---

**Definition:** Connecting MCP servers **directly** means the AI client loads every tool schema from each local or remote MCP server at connect time. **MCPZERO** publishes local servers through a zero-trust gateway with semantic aggregation and progressive discovery so agents search tools on demand.

## At a glance

| | Direct MCP | MCPZERO |
|---|------------|---------|
| **Tool discovery** | Full `tools/list` for every server at connect | Endpoint root returns `meta_search` + `meta_call_tool` only |
| **Context / tokens** | All schemas loaded upfront | Progressive discovery — search by intent first |
| **Multiple servers** | One URL or config entry per server | One endpoint root multiplexes many backends |
| **Remote access** | Requires exposing ports, TLS, or VPN | Outbound tunnel — no inbound ports |
| **Auth at edge** | Often none or client-side only | `Authorization: Bearer` API keys enforced at gateway |
| **Upstream credentials** | May live in client config | Stay on your machine; gateway never stores them |
| **Audit** | Usually none | Per-call ledger (tool, latency, status; payload by plan) |
| **Team sharing** | Share config files or credentials | Shared endpoints; per-member API keys (Team+) |

## When direct MCP is enough

- Single local stdio server on the same machine as the client
- You want the full tool catalog visible immediately with no meta layer
- No remote access, no team, no audit requirements

## When to use MCPZERO

- **Remote AI clients** (Cursor cloud, Codex, other machines) need HTTPS access to your local MCP servers
- **Multiple MCP servers** should appear as one endpoint with intelligent routing
- **Token efficiency** matters — progressive discovery avoids loading dozens of tool schemas upfront
- **Zero-trust auth** and **audit logs** are required before exposing tools
- **Teams** need governed sharing without copying upstream credentials

## How MCPZERO changes the client URL

Instead of pointing Cursor at each local server:

```json
{
  "mcpServers": {
    "filesystem": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"] },
    "postgres": { "command": "npx", "args": ["-y", "@modelcontextprotocol/server-postgres", "..."] }
  }
}
```

You tunnel once and connect at the **endpoint root**:

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/ep_abc123` |
| **Header** | `Authorization: Bearer mz_live_…` |

The meta server routes calls to the correct backend after `meta_search` matches intent.

## Also compare

- [Comparison hub](/docs/compare/)
- [ngrok vs Microsoft vs MCPZERO](/docs/compare/mcp-gateways-2026/)
- [vs tunnel-only tools](/docs/compare/vs-tunnel-only/)
- [vs ngrok](/docs/compare/vs-ngrok/)

## Next steps

- [Getting started](/docs/getting-started/overview/)
- [Progressive discovery](/docs/gateway/progressive-discovery/)
- [Security](/docs/gateway/security/)
- [Multi-server Cursor setup](/docs/use-cases/multi-server-cursor/)
