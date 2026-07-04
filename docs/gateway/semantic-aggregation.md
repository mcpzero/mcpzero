---
title: Semantic aggregation
description: Combine multiple MCP servers behind a single endpoint with intelligent routing.
---

MCPZERO **semantic aggregation** lets you expose many local MCP servers through one endpoint. Clients can address each server directly or use the endpoint root as a unified meta server.

## How it works

When a tunnel registers one or more named servers, the gateway synthesizes an **aggregator** at the endpoint root URL:

```
https://gw.mcpzero.io/v1/ep_abc123          ← meta server (aggregator)
https://gw.mcpzero.io/v1/ep_abc123/postgres ← direct route to one server
https://gw.mcpzero.io/v1/ep_abc123/filesystem
```

Single-server tunnels started with `--mcp-cmd` or `--mcp-url` (without `--mcp-config`) register under the reserved name `default`:

```
https://gw.mcpzero.io/v1/ep_abc123          ← meta server
https://gw.mcpzero.io/v1/ep_abc123/default  ← direct route
```

A `--mcp-config` tunnel with only one configured server keeps that server's name — there is no `/default` path.

Each backend server keeps its own path. The root URL exposes meta-tools (`meta_search`, `meta_call_tool`) and server profiles so agents can discover and invoke tools without loading every schema upfront — including when there is only one backend with many tools.

## Start a multiplexed tunnel

Use `--mcp-auto` to read your existing Cursor, Claude Desktop, or Codex `mcp.json` and publish every stdio server through one tunnel:

```bash
mcpzero login
mcpzero tunnel start --endpoint ep_abc123 --mcp-auto
```

The CLI detects configured servers and registers them with the gateway. Each server gets its own public HTTP/SSE path automatically.

You can also add servers manually in the [Dashboard](/app/endpoints) or proxy HTTP upstreams alongside stdio servers on the same endpoint.

## Direct vs. aggregated access

| URL pattern | Behavior |
|-------------|----------|
| `/v1/ep_abc123/<server>` | Routes directly to one backend server (name from `--mcp-config` or multiplex). |
| `/v1/ep_abc123/default` | Direct route only for single-server tunnels started with `--mcp-cmd` / `--mcp-url` (no `--mcp-config`). |
| `/v1/ep_abc123` (root) | **Meta server** with semantic aggregation and progressive discovery. Use the named path above for direct backend access. |

## Meta-tools

When aggregation is active, the endpoint root exposes:

- `resources/list` + `resources/read` — markdown profiles for each backend server
- `meta_search({ intent, context?, limit? })` — match intent to concrete backend tools
- `meta_call_tool({ server, tool, arguments? })` — invoke a matched tool on the correct backend

See [Progressive discovery](/docs/gateway/progressive-discovery/) for how clients should integrate with the meta server.

## Cross-endpoint aggregation (Team+)

On **Team** and **Enterprise** plans, MCPZERO can aggregate tools and discovery across **multiple endpoints** — not just multiple servers within one endpoint. This lets teams compose a governed MCP surface from several tunnels and environments.

Cross-endpoint aggregation requires the Team plan or above. See [Plans & pricing](/docs/pricing/plans/).

## Next

- [Progressive discovery](/docs/gateway/progressive-discovery/)
- [Tunnel reference](/docs/cli/tunnel/)
- [Cursor setup](/docs/cli/cursor/)
