---
title: MCP endpoints
description: Call MCP tools remotely through the MCPZERO gateway.
---

Buyers (Cursor, scripts, other agents) invoke MCP tools by posting JSON-RPC to the gateway.

## Endpoint

```
POST https://gw.mcpzero.io/v1/ep_abc123/postgres
Content-Type: application/json
Authorization: Bearer mz_live_…
```

### Local development

```
POST http://localhost:8787/v1/ep_dev/postgres
Authorization: Bearer dev_key_change_me
```

## Authentication

Provide your API key via the `Authorization` header:

```
Authorization: Bearer mz_live_…
```

Keys are created in [Dashboard → API Keys](/app/api-keys). The gateway resolves the buyer from the key hash stored in D1.

## Request body

Standard JSON-RPC 2.0. Example — list tools:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list",
  "params": {}
}
```

Example — call a tool:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "read_text_file",
    "arguments": { "path": "/private/tmp/example.txt" }
  }
}
```

## Responses

- **200** — JSON-RPC result or error in body (check `error` field)
- **401** — Missing or invalid API key
- **404** — Unknown endpoint ID
- **503** — Tunnel offline (CLI not connected)

### Unauthorized example

```json
{
  "jsonrpc": "2.0",
  "error": { "code": -32001, "message": "Unauthorized" },
  "id": null
}
```

## Tunnel requirement

The target endpoint must have an **active CLI tunnel** (`mcpzero tunnel start`). The gateway forwards the JSON-RPC body over WebSocket to your local MCP server.

## Observability

Each request creates a ledger row (tool name, latency, payloads, buyer ID). View traces in [Dashboard → Ledger](/app/ledger).

## curl example

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/postgres \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

## Related

- [Cursor setup](/docs/cli/cursor/)
- [Tunnel](/docs/cli/tunnel/)
