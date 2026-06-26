---
title: Login
description: Authenticate the CLI with your MCPZERO account using a browser flow.
---

`mcpzero login` opens your browser, completes Dashboard authentication, and saves a **refresh token** locally. After login, `tunnel start` does not require `--mgmt-key`.

## Quick start

```bash
mcpzero login
mcpzero whoami
```

Credentials are stored at:

```
~/.config/mcpzero/credentials.json
```

File permissions are `0600`. The refresh token is valid for **90 days** (rolling).

## Local development

When running the web worker locally on port 8788:

```bash
mcpzero login \
  --web-base http://localhost:8788 \
  --gw-base http://localhost:8787
```

You must register an account first at [localhost:8788/app/register](http://localhost:8788/app/register).

## How it works

1. CLI starts a localhost callback server on a random port
2. Browser opens `/app/cli-auth?state=…&port=…`
3. After Lucia login, the web app redirects to `http://127.0.0.1:{port}/callback?code=…`
4. CLI exchanges the one-time code for a refresh token via `POST /app/api/cli/token`
5. Token is saved locally; browser shows “Login successful — close this window”

## Commands

| Command | Description |
|---------|-------------|
| `mcpzero login` | Browser login |
| `mcpzero whoami` | Show saved user |
| `mcpzero logout` | Delete local credentials |

## No browser (containers / remote shells)

If the CLI runs somewhere without a usable browser — a Docker container, an SSH
session, etc. — the default callback (`http://127.0.0.1:{port}/callback`) can't
reach the CLI. Use the manual flow:

```bash
mcpzero login --no-browser
```

The CLI prints a URL to open on any machine, then waits for you to paste the
resulting callback URL (or `code=` value) back into the terminal. See
[Running in a container](/docs/cli/containers/) for the full walkthrough, the
`docker run --network host` auto-callback option, and the `ca-certificates`
prerequisite.

## CI / headless environments

Use a **management key** instead of browser login. Pass it with `--mgmt-key`, or
set `MCPZERO_MGMT_KEY` in the environment:

```bash
mcpzero tunnel start \
  --endpoint ep_abc \
  --mgmt-key mzm_your_management_key \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp"
```

Create a management key in the Dashboard under **Management Keys** (shown once).
A single key can register a tunnel for any endpoint you own.

## Next

- [Start a tunnel](/docs/cli/tunnel/)
