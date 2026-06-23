---
title: Login
description: Authenticate the CLI with your MCPZERO account using a browser flow.
---

`mcpzero login` opens your browser, completes Dashboard authentication, and saves a **refresh token** locally. After login, `tunnel start` does not require `--token`.

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

## CI / headless environments

Use an **endpoint tunnel token** instead of browser login:

```bash
mcpzero tunnel start \
  --endpoint ep_abc \
  --token tt_your_tunnel_token \
  --mcp-cmd "npx -y @modelcontextprotocol/server-filesystem /tmp"
```

Copy the tunnel token from Dashboard when creating an endpoint (shown once).

## Next

- [Start a tunnel](/docs/cli/tunnel/)
