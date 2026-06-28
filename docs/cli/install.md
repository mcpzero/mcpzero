---
title: Install
description: Download and install the mcpzero CLI binary for macOS, Linux, and Windows.
---

The `mcpzero` CLI is distributed as a **pre-built binary**. You do not need Go installed.

## Requirements

The CLI needs **root CA certificates** to reach `mcpzero.io` over HTTPS. Desktop
macOS and most Linux distros already have them, but **minimal container images do
not** — there, install `ca-certificates` first. See
[Running in a container](/docs/cli/containers/) for details.

## Get started

Install the latest CLI for macOS or Linux with a single command:

```bash
curl -fsSL https://mcpzero.io/install-cli.sh | sh
```

This detects your platform, downloads the matching binary, verifies its
checksum, and installs `mcpzero` to `~/.local/bin` (falling back to
`/usr/local/bin`). Then:

```bash
mcpzero version
mcpzero login
```

<details>
<summary>Install options</summary>

```bash
# Pin a specific version
curl -fsSL https://mcpzero.io/install-cli.sh | MCPZERO_VERSION=0.1.0 sh

# Install to a custom directory
curl -fsSL https://mcpzero.io/install-cli.sh | MCPZERO_INSTALL_DIR=$HOME/bin sh

# Inspect the script before running it
curl -fsSL https://mcpzero.io/install-cli.sh -o install-cli.sh
less install-cli.sh && sh install-cli.sh
```

</details>

### Homebrew (macOS / Linux)

```bash
brew install mcpzero/tap/mcpzero-cli
```

### npm

If you already use Node.js, install the CLI globally from npm. This downloads
the prebuilt binary for your platform and puts `mcpzero` on your PATH.

```bash
npm install -g mcpzero-cli
mcpzero version
```

### Python (pip / uv / pipx)

The CLI is also published to PyPI as `mcpzero-cli`. The Python package is a thin
launcher that fetches the prebuilt binary for your platform — no compilation
needed.

```bash
# pipx (recommended — installs into an isolated environment)
pipx install mcpzero-cli

# uv
uv tool install mcpzero-cli

# pip
pip install mcpzero-cli
```

Then:

```bash
mcpzero version
```

## Manual download

Prefer to grab a binary yourself? Every build is published at
`https://mcpzero.io/dl/`. The rolling latest lives under `/dl/latest/`, and each
release is also pinned under `/dl/v<version>/`.

| Platform | Latest archive |
|----------|----------------|
| macOS Apple Silicon | [`/dl/latest/mcpzero-cli_darwin_arm64.tar.gz`](https://mcpzero.io/dl/latest/mcpzero-cli_darwin_arm64.tar.gz) |
| macOS Intel | [`/dl/latest/mcpzero-cli_darwin_amd64.tar.gz`](https://mcpzero.io/dl/latest/mcpzero-cli_darwin_amd64.tar.gz) |
| Linux ARM64 | [`/dl/latest/mcpzero-cli_linux_arm64.tar.gz`](https://mcpzero.io/dl/latest/mcpzero-cli_linux_arm64.tar.gz) |
| Linux x86_64 | [`/dl/latest/mcpzero-cli_linux_amd64.tar.gz`](https://mcpzero.io/dl/latest/mcpzero-cli_linux_amd64.tar.gz) |
| Windows x86_64 | [`/dl/latest/mcpzero-cli_windows_amd64.zip`](https://mcpzero.io/dl/latest/mcpzero-cli_windows_amd64.zip) |

Checksums are at [`/dl/latest/SHA256SUMS`](https://mcpzero.io/dl/latest/SHA256SUMS); the current version string is at [`/dl/latest/VERSION`](https://mcpzero.io/dl/latest/VERSION).

Then extract, make it executable, and move it onto your PATH:

```bash
curl -fsSL https://mcpzero.io/dl/latest/mcpzero-cli_darwin_arm64.tar.gz | tar -xz
chmod +x mcpzero
sudo mv mcpzero /usr/local/bin/mcpzero
mcpzero version
```

## Build from source (developers)

If you have the private monorepo:

```bash
make -C cli dist-macos   # or dist-linux-arm64
./cli/dist/mcpzero-darwin-arm64 version
```

Docker-based cross-compile is supported — see the CLI `Makefile`.

## Verify

```bash
mcpzero version
# mcpzero 0.0.0-dev
```

## Next

- [Login with your browser](/docs/cli/login/)
- [Start a tunnel](/docs/cli/tunnel/)
