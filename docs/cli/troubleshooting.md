---
title: Troubleshooting
description: Common CLI and tunnel issues.
---

## `command not found: mcpzero`

Ensure the binary is on your `PATH`:

```bash
chmod +x ./mcpzero-darwin-arm64
sudo mv ./mcpzero-darwin-arm64 /usr/local/bin/mcpzero
```

## Login times out

- Confirm the web app is reachable (`https://mcpzero.io` or your `--web-base`)
- Complete login in the browser tab that opened
- Disable VPN/proxy blocking `127.0.0.1` callbacks
- Retry: `mcpzero logout && mcpzero login`

## `register rejected: invalid_token`

- **Logged in?** Run `mcpzero whoami`. The endpoint must belong to your user.
- **Using `--token`?** Regenerate the tunnel token in Dashboard if expired or rotated.
- **Wrong endpoint ID?** Copy the ID from [Dashboard → Endpoints](/app/endpoints).

## `register rejected: endpoint_not_owned`

The CLI refresh token user does not own this endpoint. Create a new endpoint under your account or login as the correct user.

## `tunnel_offline` / 503 from gateway

- CLI process is not running or crashed — restart `tunnel start`
- Check CLI stderr for `tunnel registered` and `mcp session initialized`
- Only one tunnel per endpoint; another machine may have replaced your connection

## MCP calls hang or timeout

- Ensure `--mcp-cmd` starts a **stdio** MCP server (not HTTP-only)
- Wait for CLI message `mcp session initialized` before sending client requests
- On macOS filesystem MCP, use `/private/tmp/...` paths for allowed directories

## `401 Unauthorized` from gateway (API key)

- Verify the auth header: `Authorization: Bearer mz_live_…`
- Regenerate key in Dashboard if revoked
- Local dev: use `dev_key_change_me` with gateway `.dev.vars` seed

## Cursor sees tools but calls fail

- Check [Ledger](/app/ledger) for `auth_denied`, `tunnel_offline`, or `mcp_error`
- Open the trace detail page for request/response payloads

## Still stuck?

- Review [Overview](/docs/getting-started/overview/) architecture
- Local E2E checklist: repo `docs/phase1-e2e.md`
