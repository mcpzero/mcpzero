---
title: Progressive discovery
description: Let AI clients discover MCP tools on demand instead of loading full tool lists upfront.
---

**Progressive discovery** is MCPZERO's approach to tool loading for multiplexed endpoints. Instead of dumping every tool schema into the LLM context at session start, the client discovers tools on demand through a small meta-tool surface.

## The problem

A multiplexed endpoint might expose dozens of tools across postgres, filesystem, puppeteer, and custom servers. Traditional MCP clients call `tools/list` once and inject every schema into the prompt — wasting context tokens and slowing the first turn.

Progressive discovery keeps the initial tool surface small. The model learns what servers exist from lightweight profiles, then searches for specific tools only when needed.

## How it works

Point your client at the **endpoint root** (no trailing server name):

```
https://gw.mcpzero.io/v1/ep_abc123
```

When the endpoint multiplexes two or more servers, the gateway exposes a **meta server**:

| Method | Purpose |
|--------|---------|
| `resources/list` + `resources/read` | Skill-style markdown profiles for each backend server |
| `meta_search({ intent, context?, limit? })` | Gateway matches your intent to concrete backend tools |
| `meta_call_tool({ server, tool, arguments? })` | Invoke a matched tool; gateway routes to the correct backend |

The AI client never needs the full tool list up front — only server profiles and the two meta-tools. Individual servers remain directly addressable at `/v1/ep_abc123/<server>`.

## Client integration pattern

A progressive client (as used by Cursor/Codex-style agents) follows this loop:

1. Connect to the endpoint root.
2. Read server profiles from `resources/list`.
3. When the model needs a capability, call `meta_search` with a natural-language intent.
4. Register matched `server__tool` names into the LLM tools channel (schemas stay out of messages).
5. When the model calls a discovered tool by name, translate to `meta_call_tool` for gateway routing.

This pattern dramatically reduces initial token load compared to loading all tools via `tools/list` on every server path.

## Single-server endpoints

If an endpoint exposes only one server, the root URL routes straight through — no meta server. Progressive discovery applies when aggregation is active (2+ servers).

## Semantic search backend

`meta_search` and server profiles use an LLM on the gateway (OpenRouter) for intent matching. Without an API key configured, the gateway falls back to deterministic keyword matching.

> Server profiles and `meta_search` require `OPENROUTER_API_KEY` on the gateway. See deployment docs for configuration.

## Availability by plan

Progressive discovery within a single endpoint is available on **all plans** (Free and above).

Cross-endpoint progressive discovery (discovering tools across multiple endpoints) requires **Team** or **Enterprise**. See [Plans & pricing](/docs/pricing/plans/).

## Next

- [Semantic aggregation](/docs/gateway/semantic-aggregation/)
- [Cursor setup](/docs/cli/cursor/)
- [Security](/docs/gateway/security/)
