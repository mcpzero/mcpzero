---
title: Multiple MCP servers in Cursor
description: Aggregate filesystem, browser, database, and custom MCP servers behind one Cursor endpoint with progressive discovery.
---

**Use case:** You run several local MCP servers (filesystem, Playwright, postgres, custom tools) and want Cursor to reach them remotely through one secure URL — without loading every tool schema at connect time.

## Why MCPZERO fits

| Challenge | MCPZERO approach |
|-----------|------------------|
| Many servers, many configs | One `mcp.json` → one tunnel → one endpoint root |
| Context bloat from tool schemas | Progressive discovery at endpoint root |
| Remote access from Cursor | HTTPS gateway with API key auth |
| Credentials on laptop only | Upstream secrets never leave your machine |

## Architecture

```
Cursor ──HTTPS──▶ gw.mcpzero.io/v1/ep_abc123 ──WS──▶ mcpzero tunnel ──stdio──▶ mcp.json servers
                         (meta server)                                              ├── filesystem
                                                                                   ├── playwright
                                                                                   └── postgres
```

## Steps

### 1. Prepare mcp.json

Use your existing Cursor MCP config or start from the [quickstart example](https://github.com/mcpzero/mcpzero/blob/main/examples/quickstart/mcp.json):

```json
{
  "mcpServers": {
    "playwright-headless": {
      "command": "npx",
      "args": ["@playwright/mcp@latest"]
    },
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed/dir"]
    }
  }
}
```

Validate before tunneling:

```bash
mcpzero servers --mcp-config ./mcp.json
```

### 2. Start the tunnel

```bash
mcpzero login
mcpzero tunnel start --endpoint ep_abc123 --mcp-config ./mcp.json
```

Or use the interactive wizard: `mcpzero init`.

### 3. Connect Cursor at the endpoint root

Point Cursor at the **meta server** (recommended):

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/ep_abc123` |
| **Header** | `Authorization: Bearer mz_live_…` |

At the root, `tools/list` returns `meta_search` and `meta_call_tool`. The agent searches by intent, then invokes the matched tool on the correct backend.

### 4. Direct server URLs (optional)

For debugging or clients without meta-tool support, each server keeps its own path:

```
https://gw.mcpzero.io/v1/ep_abc123/filesystem
https://gw.mcpzero.io/v1/ep_abc123/playwright-headless
```

See [Semantic aggregation](/docs/gateway/semantic-aggregation/) for when to use root vs server URL.

## Verify

- [Dashboard → Activity](/app/activity) shows tool calls with latency and status
- Run `mcpzero doctor` if the tunnel or client connection fails

## Related

- [Cursor setup](/docs/cli/cursor/)
- [Progressive discovery](/docs/gateway/progressive-discovery/)
- [MCPZERO vs direct MCP](/docs/compare/vs-direct-mcp/)
