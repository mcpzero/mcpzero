---
title: Running in a container
description: Install, authenticate, and tunnel with the mcpzero CLI inside Docker containers, CI, and remote shells.
---

Running `mcpzero` inside a Docker container, CI runner, or remote shell works the
same as on a laptop — with two gotchas that don't exist on a normal desktop:

1. **No root CA certificates.** Minimal images can't verify TLS to `mcpzero.io`.
2. **No browser for login.** The default browser callback can't reach the CLI
   inside the container.

This page is the one-stop guide for both.

## 1. Install CA certificates (required)

The CLI talks to `mcpzero.io` and `gw.mcpzero.io` over HTTPS/WSS, so the system
needs a root CA bundle. Most minimal images (e.g. `ubuntu:22.04`, `alpine`) ship
without one, and every TLS call then fails with
`tls: failed to verify certificate`.

```bash
# Debian/Ubuntu
apt-get update && apt-get install -y ca-certificates
# Alpine
apk add --no-cache ca-certificates
# RHEL/Fedora
dnf install -y ca-certificates
```

This is needed for **login, token refresh, and `tunnel start`** alike — not just
login.

## 2. Install the CLI

```bash
curl -fsSL https://mcpzero.io/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
```

See [Install](/docs/cli/install/) for options and PATH details.

## 3. Authenticate

Pick the approach that fits your environment.

### Interactive: `--no-browser`

A container has no browser, and the default callback
(`http://127.0.0.1:{port}/callback`) points at the *host's* loopback, not the
container — so the normal flow can't complete. Use the manual flow:

```bash
mcpzero login --no-browser
```

1. Open the printed URL in a browser on **any** machine (e.g. your laptop).
2. Sign in. The redirect to `http://127.0.0.1:{port}/callback?code=…` **fails to
   load — that's expected.**
3. Copy the **full redirected URL** (or just the `code=` value), paste it back
   into the terminal, and press Enter.

> The one-time code expires **~60 seconds** after sign-in, so paste it promptly.
> If it times out, just re-run the command.

### Linux host: `--network host` (auto callback)

On a Linux host where the browser runs on the same machine as the Docker daemon,
start the container with host networking so the `127.0.0.1` callback reaches the
container directly — no pasting, plain `mcpzero login` works:

```bash
docker run --network host -it ubuntu:22.04
```

> This does **not** apply to Docker Desktop on macOS/Windows, where containers run
> in a VM and don't share the host's loopback. Use `--no-browser` there.

### Headless / CI: tunnel token (no login)

For unattended runs, skip browser login entirely and pass an endpoint **tunnel
token** (created once in the Dashboard):

```bash
mcpzero tunnel start \
  --endpoint ep_abc \
  --token tt_your_tunnel_token \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp"
```

## Example Dockerfile

A minimal image that installs the CA bundle and the CLI:

```dockerfile
FROM ubuntu:22.04

# Required: CA certs (TLS) + curl to fetch the installer
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

# Install the mcpzero CLI
RUN curl -fsSL https://mcpzero.io/install.sh | sh
ENV PATH="/root/.local/bin:${PATH}"

# Authenticate non-interactively at runtime with a tunnel token, e.g.:
#   docker run -e MCPZERO_TUNNEL_TOKEN=tt_... your-image
CMD ["mcpzero", "version"]
```

## Common errors

| Symptom | Fix |
|---------|-----|
| `tls: failed to verify certificate` | Install `ca-certificates` (step 1) |
| Browser callback page won't load | Use `--no-browser`, or `--network host` on Linux |
| `login timed out waiting for the pasted code` | Code expired (~60s) — re-run `mcpzero login --no-browser` |
| `command not found: mcpzero` | Add the install dir to `PATH` (step 2) |

See [Troubleshooting](/docs/cli/troubleshooting/) for the full list.
