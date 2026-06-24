# MCPZERO examples

Runnable, end-to-end examples of exposing MCP servers through
[MCPZERO](https://mcpzero.io) — both from code (SDK) and from the CLI tunnel.

> Seed for the public repo `mcpzero/examples`.

## Layout

```
examples/
  sqlite/       # expose a local SQLite database file as a remote MCP endpoint
  sdk/          # link or mirror of language SDK examples (TS / Python / Go / Rust)  [planned]
  cli-tunnel/   # exposing an existing stdio / HTTP MCP server with `mcpzero tunnel` [planned]
  clients/      # pointing Cursor / Claude Code / Codex at an MCPZERO endpoint        [planned]
```

## Available examples

- [`sqlite/`](./sqlite/) — publish a local SQLite database through the
  `mcpzero` tunnel so Cursor/Claude/`curl` can query it over a remote MCP
  endpoint, with a seed script and copy-paste commands.

## Quick links

- **SDK examples** live in the SDK repo:
  - TypeScript — `mcpzero/sdk-ts` → `typescript/examples/`
  - Python — `mcpzero/sdk-py` → `python/examples/`
- **CLI tunnel** — `mcpzero tunnel start --mcp-cmd "<your server>"`
  (see [docs/cli/tunnel](https://mcpzero.io/docs/cli/tunnel/))
- **Connect a client** — see the per-endpoint setup in the
  [dashboard](https://mcpzero.io/app/endpoints) or
  [docs](https://mcpzero.io/docs/).

## Contributing an example

Each example is a self-contained directory with its own README and a single
command to run it. Keep secrets in env vars (`MCPZERO_ENDPOINT_ID`,
`MCPZERO_MGMT_KEY`, `MCPZERO_API_KEY`).
