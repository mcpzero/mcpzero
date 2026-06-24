---
title: Rust tunnel
description: Expose an in-process or local MCP server through the MCPZERO tunnel with the mcpzero crate.
---

Expose your own MCP server through the MCPZERO tunnel directly from code — a
public Streamable-HTTP endpoint plus full dashboard visibility — without the CLI.

```toml
[dependencies]
mcpzero = "0"
# embedded HTTP proxy support:
# mcpzero = { version = "0", features = ["http-proxy"] }
```

## Prerequisites

- An [endpoint created](/app/endpoints) in the Dashboard (note its ID, e.g. `ep_abc123`)
- A [**management key**](/app/management-keys) — a user-level credential that lets
  the SDK publish (register a tunnel for) any endpoint you own, the headless
  alternative to `mcpzero login`.

Empty `Config` fields fall back to `MCPZERO_ENDPOINT_ID` / `MCPZERO_MGMT_KEY`
(gateway base from `MCPZERO_GW_BASE`, default `https://gw.mcpzero.io`).

> A management key is **not** an API key. An API key is the consumer-side
> credential AI clients use to *call* your published endpoint.

## In-process server (primary)

Dispatch each forwarded request to a `Handler` (here via `handler_fn`). Return
`Ok(None)` for a notification.

```rust
use mcpzero::{handler_fn, serve, Config, Error};
use serde_json::{json, Value};

#[tokio::main]
async fn main() -> Result<(), Error> {
    let handler = handler_fn(|req: String| async move {
        let v: Value = serde_json::from_str(&req).unwrap();
        if v.get("id").is_none() {
            return Ok(None); // notification
        }
        // route v["method"] to your server, build a JSON-RPC response …
        Ok(Some(json!({"jsonrpc":"2.0","id": v["id"], "result": {}}).to_string()))
    });

    serve(Config::default(), handler).await
}
```

## Local server (embedded proxy)

```rust
// stdio:
mcpzero::tunnel_command(Config::default(), "node", vec!["server.js".into()]).await?;

// or HTTP (requires the `http-proxy` feature):
// mcpzero::tunnel_url(Config::default(), "http://localhost:3000/mcp", vec![]).await?;
```

Both run until the gateway terminally closes the tunnel or the future is dropped.

## Visibility

In tunnel mode, visibility is produced by the gateway automatically — no extra
code is needed.

## Links

- [SDK overview](/docs/sdk/overview/)
- Crate: `mcpzero` (open source, MIT)
