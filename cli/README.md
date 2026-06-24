# mcpzero CLI

The outbound local tunnel client for **[MCPZERO](https://mcpzero.io)**. It securely exposes a
local (or private) MCP server through the MCPZERO gateway so remote AI agents can
reach it over a single outbound WebSocket — no inbound ports, no public host.

> 📖 **Full documentation lives at [mcpzero.io/docs](https://mcpzero.io/docs).**
> This README is a quick overview; the website is the source of truth.

## Install

```bash
# macOS / Linux — detects your platform, verifies the checksum, installs `mcpzero`
curl -fsSL https://mcpzero.io/install.sh | sh

# Homebrew (macOS / Linux)
brew install mcpzero/tap/mcpzero
```

Windows, pinned versions, manual downloads, and checksums are covered in the
[install guide →](https://mcpzero.io/docs/cli/install/)

## Quick start

```bash
mcpzero login          # browser login

# Expose a local stdio MCP server through a Dashboard endpoint:
mcpzero tunnel start \
  --endpoint ep_abc123 \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp"
```

`tunnel start` can expose a stdio server (`--mcp-cmd`), an HTTP server
(`--mcp-url`), multiple servers from an MCP config file (`--mcp-config`), or
servers auto-discovered from your installed AI agents (`--mcp-auto`). Run
`mcpzero tunnel start --help` for the full flag list, or read the
[tunnel guide →](https://mcpzero.io/docs/cli/tunnel/)

## Commands

| Command | Description |
|---------|-------------|
| `mcpzero login` / `logout` / `whoami` | Manage browser-based credentials |
| `mcpzero version` | Print the CLI version |
| `mcpzero tunnel start [-d]` | Start a tunnel (`-d` runs it in the background) |
| `mcpzero tunnel list` | List background tunnels |
| `mcpzero tunnel logs <id> [-f]` | Show (or follow) a tunnel's logs |
| `mcpzero tunnel attach <id>` | Follow a tunnel's logs (Ctrl-C detaches) |
| `mcpzero tunnel stop <id>` | Stop a tunnel and its child processes |
| `mcpzero tunnel rm [-f] <id>` | Remove a tunnel's records (`-f` stops it first) |

## Build from source

Builds run inside Docker, so **no local Go toolchain is required**:

```bash
make build          # single binary for the Docker host arch → dist/mcpzero
make dist           # cross-compile every supported platform → dist/
make dist-macos     # darwin/arm64 + darwin/amd64
make dist-linux     # linux/amd64 + linux/arm64
make dist-windows   # windows/amd64
make dist-checksums # SHA256SUMS for the dist/ artifacts
make test
```

Override the stamped version with `make dist VERSION=1.0.0`. Cross-compiled
binaries land in `dist/`:

| File | Platform |
|------|----------|
| `mcpzero-darwin-arm64` | macOS Apple Silicon |
| `mcpzero-darwin-amd64` | macOS Intel |
| `mcpzero-linux-amd64` | Linux x86_64 |
| `mcpzero-linux-arm64` | Linux ARM64 |
| `mcpzero-windows-amd64.exe` | Windows x86_64 |

## License

MIT — see [LICENSE](./LICENSE).
