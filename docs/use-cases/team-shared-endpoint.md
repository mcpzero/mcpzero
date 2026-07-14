---
title: Team-shared MCP endpoint
description: Share governed MCP endpoints across team members with individual API keys, tool permissions, and audit logs.
---

**Use case:** A team exposes a shared MCP endpoint (staging database tools, internal APIs, browser automation) so every member can connect from Cursor with their own credentials — without copying upstream secrets or sharing one API key.

## Why MCPZERO fits

| Challenge | MCPZERO approach |
|-----------|------------------|
| Shared tools, individual accountability | Per-member API keys with tool allow/deny |
| Credential sprawl | Upstream MCP creds stay on the tunnel host |
| Audit for compliance | Call ledger — tool, latency, status; payload retention by plan |
| Multiple environments | Separate endpoints or [endpoint clusters](/docs/gateway/endpoint-clusters/) (Team+) |

## Architecture

```
Team member A (Cursor) ──┐
Team member B (Cursor) ──┼── HTTPS + own API key ──▶ gw.mcpzero.io/v1/ep_staging
Team member C (Codex)  ──┘                                      │
                                                                WS tunnel
                                                                   ▼
                                                          Dev machine / CI runner
                                                          (mcpzero tunnel + mcp.json)
```

## Steps

### 1. Create a Team workspace

Upgrade to Team+ and invite members in [Dashboard → Team](/app/team).

### 2. One member runs the tunnel

The tunnel host keeps upstream credentials local:

```bash
mcpzero tunnel start --endpoint ep_staging --mcp-config ./mcp.json
```

Use `-d` for a detached daemon on a shared runner or developer laptop that stays online.

### 3. Issue per-member API keys

Each member creates keys in [Dashboard → API Keys](/app/api-keys). Configure tool permissions (allow/deny lists) per key when needed.

### 4. Members connect independently

Everyone uses the same endpoint URL with their own key:

| Field | Value |
|-------|-------|
| **URL** | `https://gw.mcpzero.io/v1/ep_staging` |
| **Header** | `Authorization: Bearer mz_live_<member_key>` |

### 5. Review audit logs

Admins review [Activity](/app/activity) and export NDJSON on Personal+ / Team plans.

## Endpoint clusters (Team+)

When tools live on **multiple tunnels** (e.g. staging on laptop A, prod tools on laptop B), create an **endpoint cluster** (`epc_*`) to search and call across endpoints from one URL. See [Endpoint clusters](/docs/gateway/endpoint-clusters/).

## Related

- [Team sharing](/docs/gateway/team-sharing/)
- [Security](/docs/gateway/security/)
- [Pricing](/docs/pricing/plans/)
