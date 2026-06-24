# MCPZERO Tunnel Protocol v2

This is the single source-of-truth specification for the tunnel wire protocol
that every language SDK implements. It is derived from the reference Go client
and the gateway implementation; keep all ports in sync with this document.

The tunnel lets a developer expose a local/in-process MCP server through the
MCPZERO gateway, which gives it a public Streamable-HTTP endpoint and full
dashboard visibility — without configuring domains, TLS, or the CLI.

## 1. Transport

- The SDK opens a **single WebSocket** connection to the gateway:
  - `wss://<gw-host>/tunnel/<endpointId>` (use `ws://` when the gateway base is
    `http://`, e.g. local dev).
  - The gateway base defaults to `https://gw.mcpzero.io`.
- All messages are **text frames** containing a single JSON object with a
  `"type"` discriminator.
- The connection is **long-lived**. The SDK is the dialer; the gateway is the
  acceptor. The gateway keeps **one active tunnel per endpoint** and evicts the
  previous one when a new tunnel registers (see §6).

## 2. Lifecycle

```
SDK                                   Gateway
 |  --- WS connect /tunnel/<id> ---->  |
 |  --- register ------------------->  |
 |  <-- register_ok ----------------   |   (or `error`)
 |                                     |
 |  <-- mcp_request {id, body} ------  |   (one per client call)
 |  --- mcp_stream_chunk {id, body} -> |   (zero or more)
 |  --- mcp_stream_end {id, error?} -> |   (exactly one, terminates the request)
 |                                     |
 |  --- mcp_event {body} ------------> |   (server-initiated, any time)
 |  <-- mcp_cancel {id} -------------  |   (client disconnected; abort request)
 |                                     |
 |  --- ping ------------------------> |   (every 30s)
 |  <-- pong ------------------------  |
```

## 3. Protocol version

```
PROTOCOL_VERSION       = 2
SUPPORTED_VERSIONS     = [1, 2]   // gateway accepts both; SDKs send 2
```

v1 was synchronous request/response (`mcp_request` -> `mcp_response`). v2 adds
streaming (`mcp_stream_chunk` / `mcp_stream_end`), server-initiated messages
(`mcp_event`), cancellation (`mcp_cancel`), and a `transport` hint on register.
SDKs always speak v2.

## 4. Messages

### 4.1 SDK -> Gateway

**register** — first message after connect.

```jsonc
{
  "type": "register",
  "endpointId": "ep_...",        // required
  // Authenticate with EXACTLY ONE of `auth` (preferred) or `token`:
  "auth": { "type": "management", "token": "mzm_..." }, // user-level management key
  // "token": "<tunnel-token>",  // legacy per-endpoint token from the dashboard
  "protocolVersion": 2,
  "transport": "stdio",          // hint: the upstream's wire transport
  "capabilities": ["streaming"],
  "servers": ["alpha", "beta"],  // omit/empty for a single-server tunnel
  "serverInfos": [               // same servers with per-server transport
    { "name": "alpha", "transport": "stdio" },
    { "name": "beta",  "transport": "streamable-http" }
  ]
}
```

- `transport` is one of `"stdio" | "streamable-http" | "sse"`. For an
  in-process MCP server, report `"stdio"` (it behaves like a stdio server: one
  reply per request, no out-of-band HTTP semantics).
- `servers` / `serverInfos` are only present for a multiplexed tunnel. A
  single in-process server omits both and relies on the top-level `transport`.
- Authenticate with **exactly one** credential: a user-level **management key**
  in `auth` (`{ "type": "management", "token": "mzm_..." }`, preferred) **or** a
  legacy per-endpoint **tunnel token** in the plain `token` field. The CLI may
  instead send `"auth": { "type": "cli_refresh", "token": ... }`.

**mcp_stream_chunk** — one response message for an in-flight request.

```jsonc
{ "type": "mcp_stream_chunk", "id": "<tunnel-request-id>", "body": "<json-rpc message>" }
```

**mcp_stream_end** — terminates a request. `error` set iff the upstream failed.

```jsonc
{ "type": "mcp_stream_end", "id": "<tunnel-request-id>", "error": "optional message" }
```

**mcp_event** — server-initiated message not tied to a request.

```jsonc
{ "type": "mcp_event", "body": "<json-rpc message>", "server": "alpha" }
```

**ping** — keepalive, sent every 30s.

```jsonc
{ "type": "ping" }
```

### 4.2 Gateway -> SDK

- **register_ok** `{ "type": "register_ok", "protocolVersion": 2 }` — registration accepted.
- **error** `{ "type": "error", "message": "..." }` — registration rejected (fatal) or a runtime gateway error.
- **mcp_request** `{ "type": "mcp_request", "id": "...", "body": "<json-rpc>", "server": "alpha?" }` — a client call to forward to the upstream. `id` is the **tunnel request id** (opaque, not the JSON-RPC id). `server` selects the named upstream in a multiplexed tunnel; absent => the sole upstream.
- **mcp_cancel** `{ "type": "mcp_cancel", "id": "..." }` — the originating client disconnected; abort the in-flight request `id`.
- **pong** `{ "type": "pong" }` — reply to `ping` (informational; no action required).

## 5. Request handling and the id-remap contract

`mcp_request.id` is the **tunnel request id**, distinct from the JSON-RPC `id`
inside `body`. A single upstream MCP session is multiplexed across many remote
clients, so two concurrent clients may use the **same JSON-RPC id** (e.g. both
send `id: 1`). To correlate responses correctly the bridge MUST remap ids:

1. On `mcp_request`, parse `body` as JSON-RPC.
   - If it has **no `id`** (or `id` is `null`) it is a **notification**: deliver
     to the upstream and immediately reply with `mcp_stream_end` (no chunks).
   - Otherwise it is a **request**: allocate a **unique internal id**, record
     `internalId -> { tunnelReqId, originalId }`, replace `body.id` with
     `internalId`, and deliver to the upstream.
2. When the upstream produces a message:
   - If its `id` matches a tracked `internalId`, it is the response: restore
     `body.id = originalId`, emit it as `mcp_stream_chunk` with the mapped
     `tunnelReqId`, then emit `mcp_stream_end` for that `tunnelReqId`, and drop
     the mapping.
   - If its `id` does **not** match any tracked request (or it has no `id`), it
     is **server-initiated**: emit it as `mcp_event`.
3. On `mcp_cancel`, abort the upstream work for the mapped request (if the
   transport supports it) and drop the mapping. Do not emit `mcp_stream_end`
   for a cancelled request (the gateway already abandoned it).

For the **embedded stdio proxy**, requests are serialized (write one line, read
one line) exactly like the reference stdio upstream, so id-remap is optional
there; for the **in-process transport adapter** and any concurrent transport,
id-remap is REQUIRED.

A request that fails (upstream error/exception) emits `mcp_stream_end` with
`error` set and no chunk (or after partial chunks, for streaming upstreams).

## 6. Reconnection and disconnect classification

The MCP upstream starts once and stays alive across WebSocket reconnects.

**Initial connect fails fast** — a bad endpoint/token/host surfaces immediately
rather than retrying forever.

**Reconnect backoff** (per attempt, 1-based):

```
attempt 1: 1s
attempt 2: 2s
attempt 3: 4s
attempt 4: 8s
attempt >=5: 15s (cap)
max attempts: 30, then give up
```

**Ping loop**: send `{"type":"ping"}` every 30s.

**Disconnect classification** drives whether to reconnect:

- **Terminal (stop, do NOT reconnect)**: a graceful gateway close — WebSocket
  close code `1000` (normal) or `1001` (going away). In particular, close code
  `1000` with reason `"replaced_by_new_connection"` means another tunnel
  replaced this one; reconnecting would start an endless mutual-eviction loop,
  so the SDK must stop terminally.
- **Reconnect (transient)**: abnormal closure `1006`, TLS handshake failure,
  service restart `1012`, try-again-later `1013`, or any non-close network read
  error (connection reset, etc.).
- **Fatal (stop, surface error)**: any other close code (policy `1008`,
  protocol error, auth rejection) — reconnecting won't help.

A gateway close frame is **authoritative**: if a close frame arrives, classify
from it, not from whichever read/write/ping error races first.

**Dial HTTP status mapping** (when the WS upgrade returns an HTTP response):
`4xx` (except `408` and `429`) is a fatal client error; everything else is
transient/retryable.

## 7. Authentication

- The SDK authenticates at register with **one** of:
  - a user-level **management key** in `register.auth`
    (`{ "type": "management", "token": "mzm_..." }`) — registers a tunnel for any
    endpoint the user owns (preferred); or
  - a legacy per-endpoint **tunnel token** in the `register.token` field.

  The gateway validates either by SHA-256 hash (against `management_keys` or the
  endpoint record, respectively).
- `endpointId` and the management key are obtained from the MCPZERO dashboard.
- Recommended env var convention across SDKs:
  - `MCPZERO_ENDPOINT_ID`
  - `MCPZERO_MGMT_KEY` (preferred); `MCPZERO_TUNNEL_TOKEN` (legacy)
  - `MCPZERO_GW_BASE` (optional; defaults to `https://gw.mcpzero.io`)

## 8. Visibility

In tunnel mode, **visibility is produced by the gateway automatically**: every
client call traverses `/v1/<id>` and is logged to the observability
ledger (`visibility_source: "gateway"`). The SDK does **not** need to push trace
events while tunneling — the dashboard reflects calls exactly as it does for the
CLI tunnel.

## 9. Public endpoint URLs

Once registered, clients reach the server at:

- Single server: `https://<gw>/v1/<endpointId>`
- Multiplexed:   `https://<gw>/v1/<endpointId>/<serverName>`

## 10. Compatibility policy

This protocol is a **public contract** between independently released clients
(the SDKs and the CLI) and the gateway. Treat it like a versioned API:

- **Gateway is backward compatible.** It MUST accept every protocol version in
  `SUPPORTED_VERSIONS` (currently `[1, 2]`), i.e. the current version and at
  least the previous one. A gateway deploy must never break clients already in
  the wild.
- **Clients send the highest version they implement** in `register.protocolVersion`
  and degrade gracefully if the gateway negotiates lower.
- **Changes are additive.** New message types and new fields are optional and
  ignorable by older peers. Renaming or removing a field, or changing its
  meaning, requires bumping `PROTOCOL_VERSION` — never an in-place break.
- **Deprecation needs a window.** Before dropping a version from
  `SUPPORTED_VERSIONS`, announce it and leave a grace period long enough for the
  installed CLI/SDK base to upgrade.
- **Releases are decoupled.** Because the gateway stays backward compatible, the
  gateway, each SDK, and the CLI can ship on their own schedules; a client change
  does not force a synchronized gateway change (and vice versa).

The CLI authenticates tunnels differently from the SDKs (it sends
`auth: { type: "cli_refresh", token }` from `mcpzero login`, whereas SDKs send
`auth: { type: "management", token }` or a legacy per-endpoint `token`), and it
also talks to a separate control-plane API for login. That CLI-specific contract
is documented alongside the CLI; this file is the source of truth for the tunnel
wire protocol that both share.
