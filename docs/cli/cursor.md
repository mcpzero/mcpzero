---
title: Cursor setup
description: Connect Cursor to your MCPZERO remote MCP endpoint.
---

Point Cursor's remote MCP client at your gateway URL. Cursor sends JSON-RPC over HTTP; MCPZERO forwards it through your tunnel to the local MCP server.

## 1. Start your tunnel

```bash
mcpzero tunnel start \
  --endpoint ep_abc123 \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp/mcpzero-test"
```

Confirm the endpoint shows as connected (CLI prints `tunnel registered`).

## 2. Create an API key

In [Dashboard → API Keys](/app/api-keys), generate a key. This is the credential **buyers** (including you in Cursor) use to call the endpoint.

> **Caution:** The raw key is shown **once**. Store it in your password manager or Cursor config.

## 3. Configure Cursor

In Cursor MCP settings (Remote / Streamable HTTP), add:

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/ep_abc123/postgres` |
| **Header** | `Authorization: Bearer mz_live_…` |


## 4. Test with curl

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/postgres \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

You should receive a JSON-RPC result listing tools from your MCP server.

## Endpoint root: meta server

The endpoint **root** URL `https://gw.mcpzero.io/v1/ep_abc123` (no trailing
server name) is itself a full MCP server. When your endpoint multiplexes
multiple servers, the root becomes a **meta server** for semantic aggregation
and progressive discovery.

See the dedicated guides for full details:

- [Semantic aggregation](/docs/gateway/semantic-aggregation/) — how multiple servers combine behind one endpoint
- [Progressive discovery](/docs/gateway/progressive-discovery/) — `meta_search`, `meta_call_tool`, and client integration

> Server profiles and `meta_search` use OpenRouter on the gateway. Set the
> `OPENROUTER_API_KEY` secret (`wrangler secret put OPENROUTER_API_KEY`) and
> optionally override `OPENROUTER_MODEL`. Without a key, profiles and search
> fall back to deterministic keyword matching.

## 5. Inspect calls

Open [Dashboard → Activity](/app/activity). Each Cursor tool call appears with tool name, latency, and status. Full payload retention depends on your plan — see [Security](/docs/gateway/security/).

## Next

- [Troubleshooting](/docs/cli/troubleshooting/)
