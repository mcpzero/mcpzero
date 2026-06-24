---
title: Python tunnel
description: Expose an in-process or local MCP server through the MCPZERO tunnel with the mcpzero package.
---

Expose your own MCP server through the MCPZERO tunnel directly from code — a
public Streamable-HTTP endpoint plus full dashboard visibility — without the CLI.

```bash
pip install mcpzero-sdk          # embedded proxy only
pip install "mcpzero[mcp]"   # + in-process adapter for the official MCP SDK
```

## Prerequisites

- An [endpoint created](/app/endpoints) in the Dashboard (note its ID, e.g. `ep_abc123`)
- A [**management key**](/app/management-keys) — a user-level credential that lets
  the SDK publish (register a tunnel for) any endpoint you own, the headless
  alternative to `mcpzero login`.

By convention these are read from `MCPZERO_ENDPOINT_ID` and `MCPZERO_MGMT_KEY`
(gateway base from `MCPZERO_GW_BASE`, default `https://gw.mcpzero.io`).

> A management key is **not** an API key. An API key is the consumer-side
> credential AI clients use to *call* your published endpoint.

## In-process server (primary)

Run your official `mcp` SDK server with its streams backed by the tunnel:

```python
import asyncio
import mcpzero
from mcp.server.lowlevel import Server

server = Server("my-mcp")
# @server.list_tools() / @server.call_tool() …

asyncio.run(mcpzero.serve(server, endpoint_id="ep_abc123", management_key="mzm_..."))
```

Or drive the streams yourself:

```python
async with mcpzero.tunnel(endpoint_id="ep_abc123", management_key="mzm_...") as (read, write):
    await server.run(read, write, server.create_initialization_options())
```

## Local server (embedded proxy)

A library version of `mcpzero tunnel`, for a server running as a separate
process or HTTP service:

```python
# stdio:
handle = await mcpzero.tunnel_proxy(command=["python", "server.py"])

# or HTTP:
handle = await mcpzero.tunnel_proxy(url="http://localhost:3000/mcp")

print("Live at", handle.endpoint_url)
# later:
await handle.aclose()
```

## Visibility

In tunnel mode, visibility is produced by the gateway automatically — the
dashboard reflects calls exactly as it does for the CLI tunnel. No extra code is
needed.

## Links

- [SDK overview](/docs/sdk/overview/)
- PyPI: `mcpzero` (open source, MIT)
