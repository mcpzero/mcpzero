# Expose a local SQLite database through MCPZERO

This example takes a SQLite file on your machine and publishes it as a **remote
MCP endpoint** (`https://gw.mcpzero.io/v1/<endpoint>`) without opening any inbound
port. A buyer (Cursor, Claude, Codex, `curl`, …) talks HTTP to the gateway; the
gateway forwards each call over your `mcpzero` tunnel to a local SQLite MCP
server that reads/writes the file.

```
Cursor / Claude ──HTTP──▶ gw.mcpzero.io/v1/<endpoint> ──WS──▶ mcpzero tunnel ──stdio──▶ mcp-server-sqlite ──▶ demo.db
```

The local SQLite MCP server used here is the reference
[`mcp-server-sqlite`](https://pypi.org/project/mcp-server-sqlite/), run on demand
with [`uvx`](https://docs.astral.sh/uv/). It exposes `list_tables`,
`describe_table`, `read_query`, `write_query`, and `create_table` tools.

## Prerequisites

- A registered MCPZERO account and an [endpoint](https://mcpzero.io/app/endpoints)
  (note its ID, e.g. `ep_abc123`).
- `mcpzero` CLI installed and logged in: `mcpzero login`
  (see [install docs](https://mcpzero.io/docs/cli/install/)).
- [`uv`/`uvx`](https://docs.astral.sh/uv/getting-started/installation/) on PATH
  (provides the `mcp-server-sqlite` runner). No global install needed.

## 1. Create a demo database (optional)

Skip this if you already have a `.sqlite`/`.db` file — just point `--db-path` at it.

```bash
cd examples/sqlite
sqlite3 demo.db < seed.sql      # creates demo.db with sample tables/rows
```

## 2. Start the tunnel

This launches the SQLite MCP server as a stdio subprocess and bridges it to your
endpoint. Use an **absolute path** to the database so it resolves regardless of
the working directory.

```bash
mcpzero tunnel start -d \
  --endpoint ep_abc123 \
  --mcp-config ./mcp-config.json
```

When it prints `tunnel registered`, the endpoint is online. The database file
never leaves your machine — only individual query results flow back through the
gateway.

> Run it detached as a background daemon with `-d`, and stop it later with
> `mcpzero tunnel ls` / `mcpzero tunnel rm -f <id>`.

## 3. Create an API key

In [Dashboard → API Keys](https://mcpzero.io/app/api-keys), generate a key
(`mz_live_…`). This is the credential buyers present to the gateway — it is
separate from your CLI login.

## 4. Test with curl

List the tools your SQLite server exposes:

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/sqlite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_api_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Run a read query against the demo data:

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/sqlite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_api_key" \
  -d '{
        "jsonrpc":"2.0","id":2,"method":"tools/call",
        "params":{
          "name":"read_query",
          "arguments":{"query":"SELECT name, email FROM customers LIMIT 5"}
        }
      }'
```

## 5. Connect a client

Point any remote-MCP client at the gateway URL with your API key.

| Field      | Value                                                    |
|------------|----------------------------------------------------------|
| **URL**    | `https://gw.mcpzero.io/v1/ep_abc123`                     |
| **Header** | `Authorization: Bearer mz_live_api_key` |

Cursor (`~/.cursor/mcp.json`) or any `mcpServers`-style config:

```json
{
  "mcpServers": {
    "my-sqlite": {
      "url": "https://gw.mcpzero.io/v1/ep_abc123/sqlite",
      "headers": { "Authorization": "Bearer mz_live_your_key" }
    }
  }
}
```

Every call shows up in [Dashboard → Activity](https://mcpzero.io/app/activity) with
tool name, latency, and payloads.

## Read-only / safety notes

- `mcp-server-sqlite` allows writes (`write_query`, `create_table`). To publish a
  **read-only** view, point `--db-path` at a copy, or open the file read-only by
  giving the process a copy it cannot modify (e.g. `cp demo.db ro.db && chmod 0444 ro.db`).
- The DB path and any credentials stay local; the gateway only ever sees the MCP
  JSON-RPC requests and their results.
- Keep one tunnel per endpoint — see the [tunnel docs](https://mcpzero.io/docs/cli/tunnel/).

## Local development

Against a locally running gateway (`saas/gateway` on `:8787`, seed endpoint `ep_dev`):

```bash
mcpzero login --web-base http://localhost:8788 --gw-base http://localhost:8787
mcpzero tunnel start \
  --endpoint ep_dev \
  --gw-base http://localhost:8787 \
  --mcp-cmd "uvx mcp-server-sqlite --db-path $(pwd)/demo.db"

curl -s http://localhost:8787/v1/ep_dev \
  -H "Content-Type: application/json" \
  -H "X-MCP0-Token: dev_key_change_me" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

## Alternatives

- **Node instead of Python:** `npx -y mcp-server-sqlite-npx /absolute/path/demo.db`
  in place of the `uvx` command.
- **Multiple servers over one tunnel:** put the SQLite server in an
  `mcp.json` and use `--mcp-config` (see the
  [CLI README](../../cli/README.md)).
