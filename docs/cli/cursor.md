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
server name) is itself a full MCP server, so a client can point at the endpoint
without knowing a specific server name. Its behavior depends on how many servers
the endpoint exposes:

- **Single server** — the root routes straight through to that one server (same
  as addressing it directly).
- **Multiple servers** (a multiplexed tunnel) — the root becomes a **meta
  server** for progressive discovery:
  - `resources/list` + `resources/read` — Skill-style markdown profiles for
    each backend server (LLM-generated, cached until capabilities change).
  - `meta_search({ intent, context?, limit? })` — gateway LLM matches your
    intent to concrete backend tools and returns names, schemas, and reasons.
  - `meta_call_tool({ server, tool, arguments? })` — invoke a matched tool;
    the gateway routes the call to the correct backend server.

The AI client never needs the full tool list up front — only server profiles
(from resources) and the two meta-tools. Individual servers remain directly
addressable at `https://gw.mcpzero.io/v1/ep_abc123/<server>`.

> Server profiles and meta_search use OpenRouter on the gateway. Set the
> `OPENROUTER_API_KEY` secret (`wrangler secret put OPENROUTER_API_KEY`) and
> optionally override `OPENROUTER_MODEL`. Without a key, profiles and search
> fall back to deterministic keyword matching.

## 5. Inspect calls

Open [Dashboard → Ledger](/app/ledger). Each Cursor tool call appears with tool name, latency, and payloads.

## Next

- [Troubleshooting](/docs/cli/troubleshooting/)
