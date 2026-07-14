---
title: Expose a SQLite database
description: Publish a local SQLite file as a remote MCP endpoint without opening inbound ports — query from Cursor or curl through the gateway.
---

**Use case:** You have a SQLite database on your machine and want AI clients (Cursor, Claude, Codex) or API consumers to query it remotely — without hosting the file in the cloud or opening inbound ports.

## Why MCPZERO fits

| Challenge | MCPZERO approach |
|-----------|------------------|
| Local `.db` file must stay local | Tunnel bridges stdio MCP server; file never uploaded |
| Remote HTTPS access | Gateway URL + API key |
| Governed access | Tool permissions and audit ledger |
| Optional read-only publishing | Point tunnel at a copy or read-only file |

## Architecture

```
Cursor / curl ──HTTPS──▶ gw.mcpzero.io/v1/ep_abc123 ──WS──▶ mcpzero tunnel ──stdio──▶ mcp-server-sqlite ──▶ demo.db
```

The SQLite MCP server ([`mcp-server-sqlite`](https://pypi.org/project/mcp-server-sqlite/)) exposes `list_tables`, `describe_table`, `read_query`, `write_query`, and `create_table`.

## Steps

### 1. Create or reuse a database

```bash
cd examples/sqlite   # in the mcpzero repo
sqlite3 demo.db < seed.sql
```

Or use any existing `.sqlite` / `.db` file with an absolute path.

### 2. Start the tunnel

```bash
mcpzero tunnel start -d \
  --endpoint ep_abc123 \
  --mcp-config ./mcp-config.json
```

Example `mcp-config.json`:

```json
{
  "mcpServers": {
    "sqlite": {
      "command": "uvx",
      "args": ["mcp-server-sqlite", "--db-path", "/absolute/path/to/demo.db"]
    }
  }
}
```

### 3. Create an API key

Generate a key in [Dashboard → API Keys](/app/api-keys) (`mz_live_…`).

### 4. Test with curl

List tools (direct server URL):

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123/sqlite \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_api_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Run a query:

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

### 5. Connect Cursor

Use the endpoint root for progressive discovery or `/sqlite` for direct tool access:

```json
{
  "mcpServers": {
    "my-sqlite": {
      "url": "https://gw.mcpzero.io/v1/ep_abc123",
      "headers": { "Authorization": "Bearer mz_live_your_key" }
    }
  }
}
```

## Safety notes

- `mcp-server-sqlite` allows writes. For read-only publishing, use a copy: `cp demo.db ro.db && chmod 0444 ro.db`.
- Database path and credentials stay on the tunnel host.
- Calls appear in [Dashboard → Activity](/app/activity).

## Full walkthrough

See the runnable example in the repo: [examples/sqlite/README.md](https://github.com/mcpzero/mcpzero/blob/main/examples/sqlite/README.md).

## Related

- [Tunnel](/docs/cli/tunnel/)
- [Security](/docs/gateway/security/)
- [Multiple MCP servers in Cursor](/docs/use-cases/multi-server-cursor/)
