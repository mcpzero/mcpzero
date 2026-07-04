---
title: Progressive discovery
description: Let AI clients discover MCP tools on demand instead of loading full tool lists upfront.
---

**Progressive discovery** is MCPZERO's approach to tool loading. Instead of dumping every tool schema into the LLM context at session start, the client discovers tools on demand through a small meta-tool surface at the **endpoint root**.

## The problem

An endpoint might expose dozens of tools on one server (filesystem, browser automation, a large custom API) or spread across several servers. Traditional MCP clients call `tools/list` on a direct server path once and inject every schema into the prompt — wasting context tokens and slowing the first turn.

Progressive discovery keeps the initial tool surface small. The model learns what backends exist from lightweight profiles, then searches for specific tools only when needed — even when there is only **one** backend server with many tools.

## Endpoint root vs. server URL

Progressive discovery is available when you connect to the **endpoint root**. Direct server paths use traditional `tools/list` behavior.

| URL | Progressive discovery | `tools/list` returns |
|-----|----------------------|----------------------|
| `https://gw.mcpzero.io/v1/ep_abc123` (root) | **Yes** — use `meta_search` → `meta_call_tool` | `meta_search`, `meta_call_tool` only |
| `https://gw.mcpzero.io/v1/ep_abc123/<server>` (direct) | **No** — full schemas loaded upfront | All tools on that backend server |

**Default recommendation:** configure Cursor and other agents at the endpoint root. Use a server URL only when you need traditional MCP behavior or for debugging.

See [Semantic aggregation](/docs/gateway/semantic-aggregation/) for the full root vs. server URL comparison.

## How it works

Point your client at the **endpoint root** (no trailing server name):

```
https://gw.mcpzero.io/v1/ep_abc123
```

When the endpoint has one or more registered servers, the gateway exposes a **meta server** at the root URL:

| Method | Purpose |
|--------|---------|
| `resources/list` + `resources/read` | Skill-style markdown profiles for each backend server |
| `meta_search({ intent, context?, limit? })` | Gateway matches your intent to concrete backend tools (within one or many servers) |
| `meta_call_tool({ server, tool, arguments? })` | Invoke a matched tool; gateway routes to the correct backend |

The AI client never needs the full tool list up front — only server profiles and the two meta-tools. Individual servers remain directly addressable at `/v1/ep_abc123/<server>`.

## Client integration pattern

A progressive client (as used by Cursor/Codex-style agents) follows this loop:

1. Connect to the endpoint root.
2. Read server profiles from `resources/list`.
3. When the model needs a capability, call `meta_search` with a natural-language intent.
4. Register matched `server__tool` names into the LLM tools channel (schemas stay out of messages).
5. When the model calls a discovered tool by name, translate to `meta_call_tool` for gateway routing.

This pattern dramatically reduces initial token load compared to loading all tools via `tools/list` on every server path. See the gateway benchmark in the MCPZERO repo (`saas/gateway/test/bench/`) for token comparisons.

## Single-server endpoints

Single-server tunnels also use the meta server at the root. This is especially useful when one MCP server exposes many tools: connect to the root for `meta_search` / `meta_call_tool`, or use the direct path when you want a traditional full `tools/list`.

## Semantic search backend

`meta_search` and server profiles use an LLM on the MCPZERO cloud gateway (OpenRouter) for intent matching. On **mcpzero.io** this is hosted for you — no API keys or deployment configuration required. Without LLM access, the gateway falls back to deterministic keyword matching.

## Availability by plan

Progressive discovery within a single endpoint is available on **all plans** (Free and above).

Cross-endpoint progressive discovery (discovering tools across multiple endpoints via an `epc_*` cluster URL) requires **Team** or **Enterprise**. See [Endpoint clusters](/docs/gateway/endpoint-clusters/).

## Next

- [Semantic aggregation](/docs/gateway/semantic-aggregation/)
- [Cursor setup](/docs/cli/cursor/)
- [Security](/docs/gateway/security/)
