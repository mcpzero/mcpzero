---
title: TypeScript tunnel
description: Expose an in-process or local MCP server through the MCPZERO tunnel with mcpzero-sdk.
---

Expose your own MCP server through the MCPZERO tunnel directly from code — a
public Streamable-HTTP endpoint plus full dashboard visibility — without the CLI.

```bash
npm install mcpzero-sdk
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

Attach `McpZeroTunnel` to your existing `@modelcontextprotocol/sdk` server as a
transport. Every client call that reaches the public endpoint is forwarded to
your server in-process; replies stream back over the tunnel.

```typescript
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { McpZeroTunnel } from "mcpzero-sdk";

const server = new Server(
  { name: "my-mcp", version: "1.0.0" },
  { capabilities: { tools: {} } },
);
// server.setRequestHandler(...) for your tools …

const tunnel = new McpZeroTunnel({
  endpointId: process.env.MCPZERO_ENDPOINT_ID,
  managementKey: process.env.MCPZERO_MGMT_KEY,
});
await server.connect(tunnel);

console.log(`Live at ${tunnel.endpointUrl}`);
```

## Local server (embedded proxy)

If your MCP server runs as a separate stdio command or HTTP server, use the
`tunnel()` primitive (a library version of `mcpzero tunnel`):

```typescript
import { tunnel } from "mcpzero-sdk";

// stdio:
const handle = await tunnel({ mcp: { command: "node", args: ["server.js"] } });

// or HTTP:
const handle = await tunnel({ mcp: { url: "http://localhost:3000/mcp" } });

console.log(`Live at ${handle.endpointUrl}`);
// later: await handle.close();
```

## Visibility

In tunnel mode, visibility is produced by the gateway automatically — no extra
code is needed. Every call that hits the public endpoint appears in the
dashboard exactly as it does for the CLI tunnel.

> **Note — One tunnel per endpoint:** The gateway keeps a **single active tunnel per endpoint** and evicts the previous
> connection when a new one registers. The SDK detects this graceful close and
> stops (it does not fight to reconnect).

## Links

- [SDK overview](/docs/sdk/overview/)
- npm: `mcpzero-sdk` (open source, MIT)
