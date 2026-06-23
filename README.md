# MCPZERO

**Expose, secure, and observe MCP servers — from your laptop to production clients.**

[Website](https://mcpzero.io) · [Docs](https://mcpzero.io/docs/) · [Dashboard](https://mcpzero.io/app)

MCPZERO gives any [Model Context Protocol](https://modelcontextprotocol.io)
server a public, authenticated Streamable-HTTP endpoint through a reverse tunnel
— plus full call visibility — without configuring domains, TLS, or hosting.

```
your MCP server (no auth) ──tunnel──▶ gw.mcpzero.io ──▶ auth ──▶ Cursor / Claude Code / Codex
                                  │
                              dashboard: auth, API keys, call ledger
```

## Get started

**Tunnel a local server with the CLI:**

```bash
brew install mcpzero/tap/mcpzero
mcpzero tunnel start --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp"
```

(or `curl -fsSL https://mcpzero.io/install.sh | sh`)

**Or tunnel from your own code with the SDK:**

```bash
npm install mcpzero-sdk          # TypeScript
pip install mcpzero-sdk          # Python
cargo add mcpzero-sdk            # Rust
go get github.com/mcpzero/sdk-go # Go
```

## Repositories

| Repo | What | License |
|------|------|---------|
| [`mcpzero`](https://github.com/mcpzero/mcpzero) | This repo — docs, examples, install script, protocol spec | MIT |
| [`cli`](https://github.com/mcpzero/cli) | `mcpzero` tunnel client (Go) | OSS |
| [`sdk-ts`](https://github.com/mcpzero/sdk-ts) | TypeScript SDK — npm `mcpzero-sdk` | MIT |
| [`sdk-go`](https://github.com/mcpzero/sdk-go) | Go SDK — `github.com/mcpzero/sdk-go` | MIT |
| [`sdk-py`](https://github.com/mcpzero/sdk-py) | Python SDK — PyPI `mcpzero-sdk` | MIT |
| [`sdk-rust`](https://github.com/mcpzero/sdk-rust) | Rust SDK — crate `mcpzero-sdk` | MIT |
| [`homebrew-tap`](https://github.com/mcpzero/homebrew-tap) | `brew install mcpzero/tap/mcpzero` | — |

## Documentation

- Product docs: https://mcpzero.io/docs (source of truth lives in [`docs/`](./docs/))
- Tunnel wire protocol: [`PROTOCOL.md`](./PROTOCOL.md)
- Runnable examples: [`examples/`](./examples/)
- Product design: [`project.md`](./project.md)

## License

Content in this repository (docs, examples, install script) is MIT — see
[LICENSE](./LICENSE).
