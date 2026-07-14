---
title: MCPZERO vs tunnel-only tools
description: How MCPZERO differs from generic reverse proxies and tunnel services like ngrok for exposing MCP servers.
---

**Definition:** A **tunnel-only** tool forwards network traffic from a public URL to a local port or process. **MCPZERO** is an MCP-aware aggregation gateway: tunnel transport plus edge auth, semantic multiplexing, progressive discovery, and audit.

## At a glance

| | Tunnel-only (e.g. ngrok, Cloudflare Tunnel) | MCPZERO |
|---|---------------------------------------------|---------|
| **Primary job** | Expose a local port or HTTP service | Publish and govern MCP servers for AI clients |
| **Protocol awareness** | TCP/HTTP passthrough | JSON-RPC MCP — routing, meta server, tool permissions |
| **Multi-server** | One tunnel per service, or manual path routing | One endpoint multiplexes many MCP servers |
| **Tool discovery** | N/A — client talks to raw MCP server | Progressive discovery via `meta_search` / `meta_call_tool` |
| **Auth** | Optional HTTP basic or none | Zero-trust API keys at gateway; upstream creds stay local |
| **Audit** | Access logs at best | MCP call ledger — tool name, latency, status, optional payloads |
| **Team model** | Share tunnel URL + secret | Dashboard endpoints, per-member keys, clusters (Team+) |
| **Setup for MCP** | Run MCP HTTP server + tunnel + TLS + auth yourself | `mcpzero tunnel start --mcp-config ./mcp.json` |

## What a tunnel-only stack looks like for MCP

To expose a local stdio MCP server with a generic tunnel you typically:

1. Wrap stdio MCP in an HTTP/SSE transport (or run an HTTP-native MCP server)
2. Start a tunnel to expose the port
3. Configure TLS (often provided by the tunnel vendor)
4. Add your own API key or OAuth layer
5. Repeat for each server — no built-in aggregation or progressive discovery
6. Build your own logging if you need audit trails

MCPZERO collapses these into one CLI + gateway:

```
local MCP servers ──stdio──▶ mcpzero tunnel ──WS──▶ gw.mcpzero.io ──▶ AI client
                              (reads mcp.json)      (auth · aggregate · audit)
```

## When a tunnel-only tool is enough

- You already run an HTTP MCP server with your own auth and only need a public URL
- Single service, no aggregation, no MCP-specific audit requirements
- You are building a custom gateway and only need raw connectivity

## When to use MCPZERO

- **stdio MCP servers** (filesystem, postgres, custom tools) need remote HTTPS access without writing a transport layer
- **Multiple servers** should share one endpoint with semantic routing
- **AI agents** should use progressive discovery instead of loading every schema
- **API key governance**, tool permissions, and **audit logs** are required
- **Teams** need shared endpoints without sharing upstream credentials

## Also compare

- [vs ngrok](/docs/compare/vs-ngrok/)
- [vs Cloudflare Tunnel](/docs/compare/vs-cloudflare-tunnel/)
- [ngrok vs Microsoft vs MCPZERO](/docs/compare/mcp-gateways-2026/)
- [Comparison hub](/docs/compare/)

## Next steps

- [Install CLI](/docs/cli/install/)
- [Semantic aggregation](/docs/gateway/semantic-aggregation/)
- [MCPZERO vs direct MCP](/docs/compare/vs-direct-mcp/)
- [FAQ](/docs/faq/)
