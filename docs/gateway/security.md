---
title: Security
description: Zero-trust authentication, audit logging, and enterprise security controls on the MCPZERO gateway.
---

MCPZERO is a **secure MCP aggregation gateway** — not just a reverse proxy. Every request passes through authenticated edge enforcement before reaching your local tools.

## Zero-trust authentication

MCPZERO supports two client authentication modes at the edge:

| Mode | Credential | Best for |
|------|------------|----------|
| **MCP OAuth 2.1** | Short-lived JWT from Dashboard consent | Cursor, Claude Code, Codex — no long-lived key in config |
| **API keys** | `Authorization: Bearer <mz_live_api_key>` | Scripts, curl, `mcpzero cursor add`, legacy clients |

Both are validated at the Cloudflare edge in under 5 milliseconds. Keys are generated in [Dashboard → API Keys](/app/api-keys). OAuth uses PRM discovery on the gateway and consent at `https://mcpzero.io/oauth/authorize`.

**Client OAuth setup:** [MCP OAuth 2.1](/docs/gateway/oauth/) — step-by-step for Cursor, Claude Code, and Codex.

Upstream MCP credentials (database URLs, Slack tokens, etc.) are supplied through the CLI or [Secret Vault](/app/key-vault) and are **never sent to the gateway** as part of client auth.

## Metadata vs. payload retention

By default, MCPZERO records **metadata only** for every tool call:

- Timestamp
- Tool name
- Latency
- Status code

Request and response bodies are forwarded in-memory. **Free**, **Personal**, and **Team** include tiered cloud payload storage (see table below). Retention is enforced by plan via the retention worker — see [Plans & pricing](/docs/pricing/plans/) for committed windows.

| Plan | Payload storage |
|------|-----------------|
| Free | 48-hour cloud storage |
| Personal | 7-day cloud storage |
| Team | 30-day cloud storage |
| Enterprise | Private BYO storage (S3 / R2 / OSS) |

Inspect calls in [Dashboard → Activity](/app/activity) (searchable on every plan). **Personal+** can export matching rows as NDJSON; **Personal+** also supports account webhook alerts (Team adds team-scoped alerts). See [Plans & pricing](/docs/pricing/plans/).

## Rate limiting

Every plan includes gateway rate limits to protect your endpoints:

| Plan | Rate limit |
|------|------------|
| Free | 30 requests / min |
| Personal | 30 requests / min |
| Team | 60 requests / min per endpoint |
| Enterprise | Custom |

## Enterprise security controls *(Roadmap)*

The following capabilities are on the **Enterprise roadmap**. Contact [sales@mcpzero.io](mailto:sales@mcpzero.io) for availability and early access.

### Semantic WAF

A content-aware firewall that understands JSON-RPC and tool schemas. Inspects every `tools/call` for malicious arguments, unsafe paths, and data-exfiltration patterns — blocking or flagging requests before they reach your server.

### Tool Hijacking Defense

Scans tool arguments and returned content for injection and jailbreak payloads (instruction overrides, exfil prompts, poisoned tool results), neutralizing them so a compromised tool cannot hijack the calling agent.

### DLP & compliance

Data-loss prevention controls for legal and compliance teams. Custom retention and audit policies, with full JSON-RPC logs streamed to your own storage.

## Loop prevention

The gateway detects routing cycles (including cross-endpoint loops) and blocks them before they cause infinite forwarding. This is a safety backstop when aggregating multiple servers or endpoints.

## Next

- [MCP OAuth 2.1](/docs/gateway/oauth/)
- [Team sharing](/docs/gateway/team-sharing/)
- [Plans & pricing](/docs/pricing/plans/)
- [Getting started](/docs/getting-started/overview/)
