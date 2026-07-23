---
title: MCP OAuth 2.1
description: Connect Cursor, Claude Code, and Codex to MCPZERO — OAuth 2.1 or API keys. Both auth modes are fully supported at the gateway.
---

MCPZERO supports **two client authentication modes** on every MCP endpoint. They work side by side — OAuth does not replace API keys.

| Mode | Credential | When to use |
|------|------------|-------------|
| **MCP OAuth 2.1** | Short-lived JWT from browser consent | Cursor, Claude Code, Codex — per-user tokens, scoped endpoints, no long-lived secret in config |
| **API key** *(still fully supported)* | `Authorization: Bearer mz_live_…` | Scripts, curl, CI, `mcpzero cursor add`, team members with their own keys, or any client that prefers a static credential |

**API keys are not deprecated.** Create and manage them in [Dashboard → API Keys](/app/api-keys). Per-key tool permissions, rate limits, and audit logging work the same whether the client sends an API key or an OAuth JWT.

The gateway checks credentials in this order on every request:

1. If `Authorization: Bearer …` looks like a JWT → validate OAuth access token (issuer, audience, expiry, endpoint allowlist).
2. Otherwise → validate as an API key (`mz_live_…`).

If you send a valid API key, the gateway **never** triggers an OAuth challenge — your client does not need to implement OAuth at all.

Use OAuth when you want browser-based consent and scoped JWTs. Use API keys when you want the simplest setup — see [Cursor setup](/docs/cli/cursor/) and the [API key mode](#api-key-mode) section below.

## How it works

| Role | Host | Purpose |
|------|------|---------|
| **Resource Server (RS)** | `https://gw.mcpzero.io` | Validates JWT or API key on every MCP request |
| **Protected Resource Metadata (PRM)** | `https://gw.mcpzero.io/.well-known/oauth-protected-resource` | Tells clients where to authenticate |
| **Authorization Server (AS)** | `https://mcpzero.io` | Issues tokens after Dashboard login + consent |
| **AS metadata** | `https://mcpzero.io/.well-known/oauth-authorization-server` | OAuth endpoints (authorize, token, register) |

Typical flow:

1. Your AI client calls `https://gw.mcpzero.io/v1/ep_abc123` without credentials.
2. Gateway returns **401** with `WWW-Authenticate` pointing at PRM.
3. Client reads PRM → discovers AS → registers via **DCR** (dynamic client registration) or uses pre-registered credentials.
4. Browser opens **consent** at `https://mcpzero.io/oauth/authorize` — sign in to MCPZERO and select which endpoints (and Team clusters) this token may access.
5. Client exchanges the authorization code (PKCE) for a JWT access token.
6. Subsequent MCP calls send `Authorization: Bearer <jwt>`.

**API key path (no OAuth):** Skip steps 2–5. Send `Authorization: Bearer mz_live_…` on the first request. The gateway authenticates the key immediately — no 401 challenge, no browser flow. See [API key mode](#api-key-mode).

**Important:** The JWT is scoped to the endpoint IDs you approve in consent. When connecting to `https://gw.mcpzero.io/v1/ep_abc123`, make sure `ep_abc123` is checked on the consent screen. For Team clusters, connect to `https://gw.mcpzero.io/v1/epc_…` and approve that cluster. API keys are not scoped via consent; access is governed by [per-key tool permissions](/docs/gateway/security/) and endpoint ownership instead.

## Prerequisites

1. **MCPZERO account** — [Sign up](/app/register) or sign in.
2. **Endpoint with active tunnel** — create an endpoint in the Dashboard, then:

   ```bash
   mcpzero login
   mcpzero tunnel start --endpoint ep_abc123 --mcp-auto
   ```

3. **Gateway URL** — use the **endpoint root** (recommended for semantic aggregation):

   | Resource | URL pattern |
   |----------|-------------|
   | Single endpoint | `https://gw.mcpzero.io/v1/ep_abc123` |
   | Team cluster | `https://gw.mcpzero.io/v1/epc_…` |

   Replace `gw.mcpzero.io` with `gw-dev.mcpzero.io` and `mcpzero.io` with `dev.mcpzero.io` on the dev stack.

## API key mode

API key authentication is **unchanged** from before OAuth shipped. You do not need OAuth to use MCPZERO.

### Create a key

1. Open [Dashboard → API Keys](/app/api-keys).
2. Create a key (`mz_live_…`). The raw value is shown once — store it securely.
3. Send it on every MCP request:

   ```http
   Authorization: Bearer mz_live_your_key
   ```

### curl example

```bash
curl -s https://gw.mcpzero.io/v1/ep_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer mz_live_your_key" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### CLI helper

```bash
mcpzero cursor add \
  --endpoint ep_abc123 \
  --api-key mz_live_…
```

This writes Cursor `mcp.json` with the `Authorization` header — no OAuth step.

### Per-client config (summary)

| Client | API key config |
|--------|----------------|
| **Cursor** | `"headers": { "Authorization": "Bearer mz_live_…" }` in `mcp.json` |
| **Claude Code** | `"headers": { "Authorization": "Bearer ${env:MCPZERO_API_KEY}" }` in `.mcp.json` |
| **Codex** | `bearer_token_env_var = "MCPZERO_API_KEY"` in `config.toml` |

OAuth and API keys can be used by different clients against the **same endpoint** at the same time. Team members can each use their own API key on a shared endpoint — see [Team sharing](/docs/gateway/team-sharing/).

## Cursor

Cursor supports OAuth for remote Streamable HTTP MCP servers. When you provide only a `url` (no `Authorization` header), Cursor detects the 401 challenge and runs the OAuth flow for you.

### 1. Add the server

**Settings → Tools & MCP → New MCP Server**, or edit `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project):

```json
{
  "mcpServers": {
    "mcpzero": {
      "url": "https://gw.mcpzero.io/v1/ep_abc123"
    }
  }
}
```

Use your real endpoint ID. For a Team cluster, use `https://gw.mcpzero.io/v1/epc_…` instead.

### 2. Connect with OAuth

1. Save the config and restart Cursor (or reload MCP servers).
2. Open the MCP server entry and click **Connect** / authenticate when prompted.
3. Sign in to MCPZERO in the browser if asked.
4. On the consent screen, select the endpoint(s) or cluster(s) this client should access — **include the endpoint whose URL you configured**.
5. Approve. Cursor stores the access token and refreshes it automatically.

Cursor uses dynamic client registration against MCPZERO's AS. Its OAuth redirect URI is:

```text
cursor://anysphere.cursor-mcp/oauth/callback
```

MCPZERO accepts this via DCR — you do not need to pre-register a Cursor OAuth app.

### 3. API key mode (fully supported)

You can keep using API keys exactly as before. Add an `Authorization` header — Cursor will **not** run OAuth when a Bearer token is present:

```json
{
  "mcpServers": {
    "mcpzero": {
      "url": "https://gw.mcpzero.io/v1/ep_abc123",
      "headers": {
        "Authorization": "Bearer mz_live_…"
      }
    }
  }
}
```

Create keys in [Dashboard → API Keys](/app/api-keys). Or run `mcpzero cursor add --endpoint ep_abc123 --api-key mz_live_…`. See [Cursor setup](/docs/cli/cursor/) for `mcpzero init` and direct server routes.

### 4. Verify

After connecting, run a tool from Cursor or test with curl using a minted token. In [Dashboard → Activity](/app/activity), calls should appear with your user identity.

## Claude Code

Claude Code uses HTTP transport for remote MCP servers. OAuth is triggered when the server returns 401 with MCP OAuth metadata.

### Option A: CLI (recommended)

```bash
claude mcp add --transport http mcpzero https://gw.mcpzero.io/v1/ep_abc123
```

Then inside Claude Code:

```text
/mcp
```

Select **mcpzero** and choose **Authenticate**. Complete browser login and consent.

Scopes: `local` (default, only you), `project` (writes `.mcp.json` for the repo), or `user` (all your projects):

```bash
claude mcp add --transport http --scope project mcpzero https://gw.mcpzero.io/v1/ep_abc123
```

### Option B: `.mcp.json`

Claude Code requires an explicit `type` for HTTP servers:

```json
{
  "mcpServers": {
    "mcpzero": {
      "type": "http",
      "url": "https://gw.mcpzero.io/v1/ep_abc123"
    }
  }
}
```

> A `url` without `type` is treated as stdio and will not work.

After adding the file, run `/mcp` → **Authenticate** for **mcpzero**.

### Static OAuth credentials (advanced)

If DCR is blocked in your environment, register a client manually:

```bash
curl -s https://mcpzero.io/oauth/register \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "claude-code",
    "redirect_uris": ["http://127.0.0.1:8080/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"]
  }'
```

Then add with pre-configured OAuth (use the `client_id` from the response; `--callback-port` must match your redirect URI):

```bash
claude mcp add --transport http \
  --client-id cli_xxxxxxxx \
  --callback-port 8080 \
  mcpzero https://gw.mcpzero.io/v1/ep_abc123
```

### API key mode (fully supported)

Skip `/mcp` authentication. Add a Bearer header instead — Claude Code uses the key directly and does not start OAuth:

```json
{
  "mcpServers": {
    "mcpzero": {
      "type": "http",
      "url": "https://gw.mcpzero.io/v1/ep_abc123",
      "headers": {
        "Authorization": "Bearer ${env:MCPZERO_API_KEY}"
      }
    }
  }
}
```

Set `MCPZERO_API_KEY` in your shell profile. Keys are created in [Dashboard → API Keys](/app/api-keys).

## Codex CLI

OpenAI **Codex CLI** (and the ChatGPT desktop app when sharing the same config) stores MCP servers in `~/.codex/config.toml` (or project `.codex/config.toml`).

### 1. Add the server

```toml
[mcp_servers.mcpzero]
url = "https://gw.mcpzero.io/v1/ep_abc123"
scopes = ["mcp:tools", "mcp:resources", "mcp:prompts"]
```

Or via CLI:

```bash
codex mcp add mcpzero --url https://gw.mcpzero.io/v1/ep_abc123
```

### 2. Log in with OAuth

```bash
codex mcp login mcpzero
```

Codex opens a browser, completes PKCE against MCPZERO's AS, and stores credentials in the OS keyring (configurable via `mcp_oauth_credentials_store`).

### 3. Callback port (if required)

Some environments need a fixed localhost callback:

```toml
mcp_oauth_callback_port = 5555
```

Codex appends a server-specific suffix to `mcp_oauth_callback_url` when set — register the **full derived redirect URI** if you use a custom callback URL. For typical local use, the default ephemeral port works with MCPZERO DCR.

### 4. API key mode (fully supported)

Use a bearer token from an environment variable — no `codex mcp login` required:

```toml
[mcp_servers.mcpzero]
url = "https://gw.mcpzero.io/v1/ep_abc123"
bearer_token_env_var = "MCPZERO_API_KEY"
```

Export `MCPZERO_API_KEY=mz_live_…` before starting Codex. Create keys in [Dashboard → API Keys](/app/api-keys).

## Endpoint clusters (Team+)

Team clusters (`epc_*`) work the same way — point the client at the cluster root:

```text
https://gw.mcpzero.io/v1/epc_2b94b1088a
```

During OAuth consent, check the **Team clusters** section and select the cluster. The JWT audience includes the cluster URL. Each member endpoint in the cluster must have an active tunnel.

**API keys:** clusters also accept `Authorization: Bearer mz_live_…` — no OAuth consent step. Team plan and cluster ACL still apply. See [Endpoint clusters](/docs/gateway/endpoint-clusters/).

## Discovery reference

Verify OAuth metadata (production):

```bash
# Protected Resource Metadata (gateway)
curl -s https://gw.mcpzero.io/.well-known/oauth-protected-resource | jq

# Authorization Server metadata (dashboard)
curl -s https://mcpzero.io/.well-known/oauth-authorization-server | jq

# Unauthenticated MCP call → 401 challenge
curl -si https://gw.mcpzero.io/v1/ep_abc123 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  | head -20
```

Expected PRM fields include `authorization_servers: ["https://mcpzero.io"]` and scopes `mcp:tools`, `mcp:resources`, `mcp:prompts`.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| **401 after OAuth** | Consent did not include the endpoint ID in the URL. Re-authenticate and check the right `ep_*` or `epc_*`. |
| **403 Team plan required** (clusters) | Endpoint clusters require Team+. Upgrade or use a single `ep_*` endpoint. |
| **Tunnel not connected** | Run `mcpzero tunnel start` and confirm the endpoint shows connected in the Dashboard. |
| **Browser does not open** | Copy the authorize URL from the terminal (Claude Code `/mcp`) or run `codex mcp login` again. |
| **Invalid redirect URI** | Redirect URI must match DCR exactly. Re-register the client or let the agent use DCR with its native callback URI. |
| **Want API keys instead of OAuth** | Add `Authorization: Bearer mz_live_…` (Cursor / Claude headers) or `bearer_token_env_var` (Codex). No OAuth flow runs when a valid key is sent. See [API key mode](#api-key-mode). |

Upstream integrations (Slack, Google) use **Secret Vault** OAuth — connect via Dashboard browser flow (or paste fallback). The gateway refreshes access tokens near expiry when provider client credentials are configured. See [Security](/docs/gateway/security/) and [Dashboard → Key Vault](/app/key-vault).

## Next

- [Cursor setup](/docs/cli/cursor/) — API keys, `mcpzero init`, direct routes
- [Security](/docs/gateway/security/) — auth, audit, rate limits
- [Endpoint clusters](/docs/gateway/endpoint-clusters/) — Team+ cross-endpoint meta server
- [Team sharing](/docs/gateway/team-sharing/) — shared endpoints for members
