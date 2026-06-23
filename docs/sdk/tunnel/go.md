---
title: Go tunnel
description: Expose an in-process or local MCP server through the MCPZERO tunnel with the Go SDK.
---

Expose your own MCP server through the MCPZERO tunnel directly from code — a
public Streamable-HTTP endpoint plus full dashboard visibility — without the CLI.

```bash
go get github.com/mcpzero/sdk-go
```

## Prerequisites

- An [endpoint created](/app/endpoints) in the Dashboard (note its ID, e.g. `ep_abc123`)
- The endpoint's **tunnel token**

`Config` fields fall back to `MCPZERO_ENDPOINT_ID` / `MCPZERO_TUNNEL_TOKEN`
(gateway base from `MCPZERO_GW_BASE`, default `https://gw.mcpzero.io`).

## In-process server (primary)

Wrap your server's JSON-RPC request handling in a `HandlerFunc`. Return
`(nil, nil)` for a notification.

```go
package main

import (
	"context"
	"encoding/json"

	mcpzero "github.com/mcpzero/sdk-go"
)

func main() {
	handler := func(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
		// Process one JSON-RPC request; return its JSON-RPC response.
		return myServer.HandleJSONRPC(ctx, req)
	}
	_ = mcpzero.Serve(context.Background(), mcpzero.Config{}, handler)
}
```

## Local server (embedded proxy)

```go
// stdio:
err := mcpzero.TunnelCommand(ctx, mcpzero.Config{}, "node", "server.js")

// or HTTP:
err := mcpzero.TunnelURL(ctx, mcpzero.Config{}, "http://localhost:3000/mcp", nil)
```

Both block until `ctx` is cancelled or the gateway terminally closes the tunnel.
The public endpoint is `cfg.EndpointURL()`.

## Visibility

In tunnel mode, visibility is produced by the gateway automatically — no extra
code is needed.

## Links

- [SDK overview](/docs/sdk/overview/)
- Module: `github.com/mcpzero/sdk-go` (open source, MIT)
