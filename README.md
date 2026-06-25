# MCPZERO

**The zero-trust security gateway for MCP — expose, secure, and observe MCP
servers from your laptop to production clients.**

[Website](https://mcpzero.io) · [Docs](https://mcpzero.io/docs/) · [Dashboard](https://mcpzero.io/app)

MCPZERO is an enterprise-grade **zero-trust gateway** for the
[Model Context Protocol](https://modelcontextprotocol.io). It puts every local
MCP server behind an authenticated, observable edge — so AI agents reach your
tools without you ever exposing domains, ports, TLS, or credentials.

It is not just a reverse proxy: MCPZERO adds an identity-aware security layer,
data-loss controls, and full call auditing in front of MCP traffic.

```
your MCP server (no auth) ──tunnel──▶ gw.mcpzero.io ──▶ zero-trust gateway ──▶ Cursor / Claude Code / Codex
                                  │                    (auth · DLP · audit)
                              dashboard: auth, API keys, call ledger
```

## Why a gateway, not a proxy

| Pillar | What it means |
|--------|---------------|
| **Zero-Config** | Reads your existing `mcp.json` and multiplexes every local stdio server through one encrypted tunnel — no domains, TLS, or hosting to manage. |
| **Zero-Trust** | Every public endpoint is enforced at the edge. Clients authenticate with `Authorization: Bearer`; auth resolves in under 5ms and the protocol surface of your tools is never exposed to the internet. |
| **Zero-Leak** | The gateway forwards in-memory and persists metadata only (tool, latency, status). Request/response bodies are never stored by default — stream full audit logs to your own S3 / R2 / OSS instead. |

## Gateway intelligence

Beyond transport and auth, MCPZERO inspects and orchestrates MCP traffic at the
edge:

- **Semantic WAF** — a content-aware firewall that understands JSON-RPC and tool
  schemas, not just HTTP. It inspects every `tools/call` for malicious arguments,
  unsafe paths, and data-exfiltration patterns, and blocks or flags requests by
  policy before they ever reach your server.
- **Tool Hijacking Defense** — scans tool arguments and returned content for
  injection and jailbreak payloads (instruction overrides, exfil prompts,
  poisoned tool results), neutralizing them so a compromised tool can't hijack
  the calling agent.
- **MCP Orchestration** — fan one endpoint out to many MCP servers with smart
  routing, health checks, failover, and per-tool access control — composing
  several local and remote servers into a single governed surface for any agent.

## Get started

**Tunnel a local server with the CLI:**

```bash
brew install mcpzero/tap/mcpzero
mcpzero tunnel start --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp"
```

(or `curl -fsSL https://mcpzero.io/install.sh | sh`, or with Go:
`go install github.com/mcpzero/mcpzero/cli/cmd/mcpzero@latest`)

The CLI source lives in [`cli/`](./cli/); see its [README](./cli/README.md) for
building from source and the full command reference.

**Or tunnel from your own code with the SDK:**

```bash
npm install mcpzero-sdk          # TypeScript
pip install mcpzero-sdk          # Python
```

## Repositories

| Repo | What | License |
|------|------|---------|
| [`mcpzero`](https://github.com/mcpzero/mcpzero) | This repo — docs, examples, install script, protocol spec, and the CLI | MIT |
| [`sdk-ts`](https://github.com/mcpzero/sdk-ts) | TypeScript SDK — npm `mcpzero-sdk` | MIT |
| [`sdk-py`](https://github.com/mcpzero/sdk-py) | Python SDK — PyPI `mcpzero-sdk` | MIT |
| [`homebrew-tap`](https://github.com/mcpzero/homebrew-tap) | `brew install mcpzero/tap/mcpzero` | — |

## Documentation

- Product docs: https://mcpzero.io/docs (source of truth lives in [`docs/`](./docs/))
- Tunnel wire protocol: [`PROTOCOL.md`](./PROTOCOL.md)
- Runnable examples: [`examples/`](./examples/)

## License

Content in this repository (docs, examples, install script) is MIT — see
[LICENSE](./LICENSE).
