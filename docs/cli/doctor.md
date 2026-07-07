---
title: Doctor
description: Diagnose MCPZERO CLI login, connectivity, and local setup.
---

`mcpzero doctor` runs a quick health check before you debug tunnel or Cursor
issues. It exits with a non-zero status when any required check fails.

## Usage

```bash
mcpzero doctor
```

## Checks

| Check | What it verifies |
|-------|------------------|
| CLI version | Installed binary version |
| Login | `~/.config/mcpzero/credentials.json` exists |
| Dashboard reachable | `web_base` responds |
| Gateway reachable | `gw_base/health` returns OK |
| CLI token valid | `GET /app/api/cli/me` succeeds |
| Endpoints | You have at least one endpoint (warn if none) |
| API keys | At least one active key (warn if none) |
| Cursor config | Global `~/.cursor/mcp.json` exists (warn if missing) |
| Background tunnels | Registered daemon tunnels (when supported) |

## Example

```text
mcpzero doctor (v0.2.1)

  [✓] CLI version: 0.2.1
  [✓] Credentials file: you@example.com
  [✓] Dashboard reachable: https://mcpzero.io
  [✓] Gateway reachable: https://gw.mcpzero.io (ok)
  [✓] CLI token valid: free plan
  [✓] Endpoints: 1 total, 0 online
  [!] API keys: 0 active
      → create a key in the dashboard or run: mcpzero init
  [✓] Cursor config: /Users/you/.cursor/mcp.json

All required checks passed (1 warning(s)).
```

## Tunnel preflight

`mcpzero tunnel start` runs a lighter preflight automatically:

- Gateway `/health` reachable
- Logged in (or `--mgmt-key` set)
- Endpoint belongs to your account
- Endpoint ID format (`ep_…`)

Skip with `--skip-preflight` when you know the environment is fine (e.g. air-gapped tests).

## Next

- [Login](/docs/cli/login/)
- [Troubleshooting](/docs/cli/troubleshooting/)
