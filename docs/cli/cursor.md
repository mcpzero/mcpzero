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

### Local development

| Field | Value |
|-------|-------|
| **URL** | `http://localhost:8787/v1/ep_dev/postgres` |
| **Header** | `Authorization: Bearer dev_key_change_me` |

## 4. Test with curl

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/postgres \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

You should receive a JSON-RPC result listing tools from your MCP server.

## 5. Inspect calls

Open [Dashboard → Ledger](/app/ledger). Each Cursor tool call appears with tool name, latency, and payloads.

## Next

- [MCP HTTP API reference](/docs/api/mcp-endpoints/)
- [Troubleshooting](/docs/cli/troubleshooting/)
