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

Or multiplex every server from your existing config:

```bash
mcpzero tunnel start --endpoint ep_abc123 --mcp-auto
```

Confirm the endpoint shows as connected (CLI prints `tunnel registered`).

## 2. Create an API key

In [Dashboard → API Keys](/app/api-keys), generate a key. This is the credential Cursor uses to call the endpoint (`mz_live_…`).

> **Caution:** The raw key is shown **once**. Store it in your password manager or Cursor config.

## 3. Configure Cursor (recommended: endpoint root)

In Cursor MCP settings (Remote / Streamable HTTP), add the **endpoint root** — the meta server for semantic aggregation and progressive discovery:

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/ep_abc123` |
| **Header** | `Authorization: Bearer mz_live_…` |

This works for single-server and multi-server endpoints. The root exposes `meta_search` and `meta_call_tool` instead of loading every backend tool schema upfront.

## 4. Test the meta server with curl

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

You should receive `meta_search` and `meta_call_tool` in the tool list.

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":2,"method":"meta_search","params":{"intent":"list files"}}'
```

## Direct route (optional)

To call one backend server with a traditional full `tools/list`, use a **server URL**:

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/ep_abc123/<server>` |
| **Header** | `Authorization: Bearer mz_live_…` |

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/filesystem \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Direct routes return that server's full tool schemas. Use for debugging or clients that do not support meta-tools.

See [Semantic aggregation](/docs/gateway/semantic-aggregation/) and [Progressive discovery](/docs/gateway/progressive-discovery/) for when to use root vs. server URLs.

## 5. Inspect calls

Open [Dashboard → Activity](/app/activity). Each Cursor tool call appears with tool name, latency, and status. Full payload retention depends on your plan — see [Security](/docs/gateway/security/).

## Next

- [Troubleshooting](/docs/cli/troubleshooting/)
