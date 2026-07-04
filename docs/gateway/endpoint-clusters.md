---
title: Endpoint clusters
description: Aggregate MCP tools and progressive discovery across multiple endpoints with epc_* cluster URLs.
---

An **endpoint cluster** (`epc_*`) is a virtual gateway surface that combines **multiple endpoints** (`ep_*`) into one meta server URL. Use it when your team runs several tunnels — staging vs. production, different developers' machines, or separate environments — and you want a single Cursor connection that can discover and call tools across all of them.

Endpoint clusters require the **Team** plan or above. See [Plans & pricing](/docs/pricing/plans/).

## Endpoint vs. cluster

| | **Endpoint** (`ep_*`) | **Endpoint cluster** (`epc_*`) |
|---|----------------------|--------------------------------|
| **What it aggregates** | Multiple MCP **servers** behind one tunnel | Multiple **endpoints**, each with its own tunnel and servers |
| **Gateway URL** | `https://gw.mcpzero.io/v1/ep_abc123` | `https://gw.mcpzero.io/v1/epc_abc123` |
| **Tunnel required** | Yes — `mcpzero tunnel start --endpoint ep_…` | No tunnel on the cluster itself; each member endpoint must have an active tunnel |
| **Plans** | All plans | Team & Enterprise |
| **Meta-tools** | `meta_search`, `meta_call_tool` | Same surface; routing includes `endpoint_id` |

Within a single endpoint, [semantic aggregation](/docs/gateway/semantic-aggregation/) already combines every server registered on that tunnel. A cluster goes one level up: it composes **several endpoints** into one governed MCP surface for agents.

## How it works

When you call a cluster root URL, the gateway exposes the same progressive-discovery contract as a single endpoint — but search and routing span every online member endpoint in the cluster:

```
https://gw.mcpzero.io/v1/epc_team_prod          ← cluster meta server
https://gw.mcpzero.io/v1/ep_staging             ← member endpoint (direct)
https://gw.mcpzero.io/v1/ep_alice_dev           ← member endpoint (direct)
https://gw.mcpzero.io/v1/ep_bob_tools/postgres  ← member server (direct)
```

The cluster meta server does not replace member endpoints. Each `ep_*` keeps its own root URL and server paths. The cluster adds a **team-wide entry point** that searches and routes across members.

## Create a cluster

1. Subscribe to **Team** (or Enterprise) and create a team in [Dashboard → Team](/app/team).
2. Open [Dashboard → Team → Clusters](/app/team/clusters).
3. **Create a cluster** — give it a name (e.g. `Production agents`). MCPZERO assigns an ID like `epc_2b94b1088a`.
4. **Add member endpoints** — on the cluster detail page, select endpoints you own. Both you and the endpoint owner must be on the same team.
5. Ensure each member endpoint has an **active tunnel** (`mcpzero tunnel start --endpoint ep_…`).
6. Copy the cluster gateway URL from the detail page:

   `https://gw.mcpzero.io/v1/epc_2b94b1088a`

Cluster **status** reflects member tunnels: online when at least one member endpoint is connected, offline when none are.

## Configure Cursor (cluster root)

Point Cursor at the **cluster root** — same pattern as a single endpoint:

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/epc_2b94b1088a` |
| **Header** | `Authorization: Bearer mz_live_…` |

Use a **team API key** (created under your team account). The key's team must match the cluster's team.

See [Cursor setup](/docs/cli/cursor/) for the full remote MCP configuration.

## Meta-tools on a cluster

At the cluster root, `tools/list` returns `meta_search` and `meta_call_tool` — not every backend schema. Offline member endpoints are skipped until their tunnels reconnect.

### `meta_search`

Search spans all online member endpoints. Each match includes the target **endpoint id** so the client can route the call:

```json
{
  "intent": "read a file from disk",
  "matches": [
    {
      "endpoint_id": "ep_0476da16c9",
      "server": "local-fs-tmp",
      "tool": "read_file",
      "description": "Read a file from the filesystem",
      "inputSchema": { "type": "object", "properties": { "path": { "type": "string" } } }
    }
  ]
}
```

### `meta_call_tool`

On a cluster, `meta_call_tool` **requires** `endpoint_id` from the search result (in addition to `server` and `tool`):

```json
{
  "endpoint_id": "ep_0476da16c9",
  "server": "local-fs-tmp",
  "tool": "read_file",
  "arguments": { "path": "/tmp/notes.txt" }
}
```

On a single endpoint (`ep_*`), `endpoint_id` is omitted — the gateway already knows which endpoint you connected to.

Progressive clients (Cursor, Codex-style agents) should register discovered tools as `server__tool` in the LLM tools channel, then translate model calls to `meta_call_tool` with the `endpoint_id` from the original `meta_search` match. See [Progressive discovery](/docs/gateway/progressive-discovery/).

### Server profiles

`resources/list` on a cluster returns one profile per **member endpoint + server**, named `ep_abc123/server-name`. Profile URIs use the form `mcpzero://cluster/ep_abc123/server-name`.

## Test with curl

```bash
curl -s https://gw.mcpzero.io/v1/epc_2b94b1088a \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

```bash
curl -s https://gw.mcpzero.io/v1/epc_2b94b1088a \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":2,"method":"meta_search","params":{"intent":"list files"}}'
```

## Activity & audit

Gateway calls to `epc_*` URLs are logged in [Dashboard → Activity](/app/activity) under the cluster id. `meta_call_tool` traces include the target member `endpoint_id` for cross-endpoint routing.

## CLI note

When you run `mcpzero tunnel start --mcp-auto`, the server list may show existing remote URLs — including cluster roots (`epc_*`) already configured in Cursor. Those are **client connection targets**, not tunnels to start. Select only the stdio or local HTTP servers you want this tunnel to publish.

## Next

- [Team sharing](/docs/gateway/team-sharing/)
- [Semantic aggregation](/docs/gateway/semantic-aggregation/)
- [Progressive discovery](/docs/gateway/progressive-discovery/)
- [Plans & pricing](/docs/pricing/plans/)
