# CLI ↔ MCPZERO contract

The single source-of-truth for everything the `mcpzero` CLI expects from the
MCPZERO backend. The CLI is an independently released client; keep this contract
versioned and backward compatible so the CLI and the backend can ship on their
own schedules (see [Compatibility](#compatibility)).

The CLI talks to **two surfaces**:

| Surface | Base (default) | Purpose |
|---------|----------------|---------|
| **Control plane** (Web/SaaS) | `https://mcpzero.io` (`--web-base`, `MCPZERO_WEB_BASE`) | Login: exchange a browser auth code for a refresh token |
| **Data plane** (gateway) | `https://gw.mcpzero.io` (`--gw-base`, `MCPZERO_GW_BASE`) | Tunnel WebSocket + public MCP endpoint |

---

## 1. Control plane — login

`mcpzero login` runs a browser-based authorization-code flow with a loopback
redirect:

1. The CLI generates a random `state` and starts a localhost HTTP server on
   `127.0.0.1:<port>` with a `/callback` handler.
2. It opens the browser to:

   ```
   GET {webBase}/app/cli-auth?state=<state>&port=<port>
   ```

3. After the user authenticates in the browser, the web app redirects to:

   ```
   GET http://127.0.0.1:<port>/callback?code=<code>&state=<state>
   ```

   The CLI verifies `state` matches what it generated.
4. The CLI exchanges the code for a refresh token:

   ```
   POST {webBase}/app/api/cli/token
   Content-Type: application/json
   Accept: application/json

   { "code": "<code>", "state": "<state>", "device_name": "<hostname>" }
   ```

   **200 OK** response body:

   ```json
   {
     "refresh_token": "<token>",
     "user_id": "<id>",
     "email": "<email>",
     "gw_base": "https://gw.mcpzero.io",
     "web_base": "https://mcpzero.io"
   }
   ```

   `gw_base` / `web_base` let the server pin the CLI to a specific deployment;
   the CLI falls back to its configured/default bases when they are empty.
   Any non-200 is treated as a login failure (body surfaced to the user).

### Credential storage

Credentials are written to `os.UserConfigDir()/mcpzero/credentials.json`
(dir `0700`, file `0600`):

```json
{ "refresh_token": "...", "user_id": "...", "email": "...", "gw_base": "...", "web_base": "..." }
```

`mcpzero logout` removes this file. The `refresh_token` is long-lived and is
the credential the CLI presents when opening a tunnel (below).

---

## 2. Data plane — tunnel

The tunnel WebSocket wire protocol (`wss://<gw>/tunnel/<endpointId>`, register /
`mcp_request` / streaming / reconnect / disconnect classification) is **identical
to the SDKs** and specified once in this repo's top-level
[`PROTOCOL.md`](../PROTOCOL.md) (tunnel protocol v2). Do not duplicate it here —
link to it.

The only CLI-specific difference is **authentication on `register`**:

- **CLI** sends the stored refresh token:

  ```json
  { "type": "register", "endpointId": "...", "protocolVersion": 2,
    "auth": { "type": "cli_refresh", "token": "<refresh_token>" }, ... }
  ```

  The gateway validates the refresh token, resolves the user, and authorizes the
  endpoint. This is why `mcpzero tunnel` does not need a per-endpoint token.

- **SDKs** instead send a per-endpoint tunnel `token` field. The gateway accepts
  either form on the same `register` message.

Everything else on the wire — `register_ok`, `mcp_request`, `mcp_stream_chunk`,
`mcp_stream_end`, `mcp_event`, `mcp_cancel`, `ping`/`pong`, close-code
classification — is the shared protocol.

---

## 3. Public endpoint URLs

After a tunnel registers, buyers reach the server through the gateway:

- Single server: `https://<gw>/v1/<endpointId>`
- Multiplexed:   `https://<gw>/v1/<endpointId>/<serverName>`

Buyers authenticate with their own API key (`X-MCP0-Token: <key>` /
`Authorization: Bearer <key>` / `X-API-Key: <key>`), independent of the CLI's
login credentials.

---

## Compatibility

Both surfaces are public contracts. The backend MUST stay backward compatible:

- **Control plane**: do not remove or repurpose existing request/response fields
  on `/app/api/cli/token`; new fields are optional and ignorable by older CLIs.
- **Data plane**: governed by the tunnel protocol's version negotiation —
  `SUPPORTED_VERSIONS` on the gateway includes the current and previous
  versions; changes are additive and bump `PROTOCOL_VERSION`. See the
  Compatibility policy section in the top-level [`PROTOCOL.md`](../PROTOCOL.md).
- Because of this, a CLI change never forces a synchronized gateway/SaaS change
  (and vice versa). Removing support for an old version requires an announced
  grace period sized to the installed CLI base.

> This file ships with the CLI in the mcpzero repo; the repo's top-level
> [`PROTOCOL.md`](../PROTOCOL.md) remains the canonical source for the shared
> tunnel wire format.
