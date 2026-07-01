---
title: Security
description: Zero-trust authentication, audit logging, and enterprise security controls on the MCPZERO gateway.
---

MCPZERO is a **secure MCP aggregation gateway** — not just a reverse proxy. Every request passes through authenticated edge enforcement before reaching your local tools.

## Zero-trust authentication

All public MCP endpoints require `Authorization: Bearer <api_key>`. Keys are generated in [Dashboard → API Keys](/app/api-keys) and validated at the Cloudflare edge in under 5 milliseconds.

Clients never see your local network, ports, or upstream credentials. Upstream URLs and auth headers are supplied entirely through the CLI and are **never sent to the gateway**.

## Metadata vs. payload retention

By default, MCPZERO records **metadata only** for every tool call:

- Timestamp
- Tool name
- Latency
- Status code

Request and response bodies are forwarded in-memory and dropped — they never hit disk on the Free tier.

| Plan | Payload storage |
|------|-----------------|
| Free | Metadata only |
| Personal | 48-hour cloud storage |
| Team | 30-day cloud storage |
| Enterprise | Private BYO storage (S3 / R2 / OSS) |

Inspect calls in [Dashboard → Activity](/app/activity). Team and Enterprise plans add searchable, exportable audit logs.

## Rate limiting

Every plan includes gateway rate limits to protect your endpoints:

| Plan | Rate limit |
|------|------------|
| Free | 30 requests / min |
| Personal | 30 requests / min |
| Team | 60 requests / min per endpoint |
| Enterprise | Custom |

## Enterprise security controls

Available on the **Enterprise** plan:

### Semantic WAF

A content-aware firewall that understands JSON-RPC and tool schemas. Inspects every `tools/call` for malicious arguments, unsafe paths, and data-exfiltration patterns — blocking or flagging requests before they reach your server.

### Tool Hijacking Defense

Scans tool arguments and returned content for injection and jailbreak payloads (instruction overrides, exfil prompts, poisoned tool results), neutralizing them so a compromised tool cannot hijack the calling agent.

### DLP & compliance

Data-loss prevention controls for legal and compliance teams. Custom retention and audit policies, with full JSON-RPC logs streamed to your own storage.

## Loop prevention

The gateway detects routing cycles (including cross-endpoint loops) and blocks them before they cause infinite forwarding. This is a safety backstop when aggregating multiple servers or endpoints.

## Next

- [Team sharing](/docs/gateway/team-sharing/)
- [Plans & pricing](/docs/pricing/plans/)
- [Getting started](/docs/getting-started/overview/)
