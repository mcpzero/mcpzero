---
title: Team sharing
description: Share MCP endpoints, tunnels, and audit visibility across your team.
---

MCPZERO lets teams **instantly share MCP capabilities** — expose local tools through governed endpoints that every member can call, inspect, and audit from a central dashboard.

## What teams share

| Resource | How it works |
|----------|--------------|
| **Endpoints** | Created in [Dashboard → Endpoints](/app/endpoints). Each endpoint has a unique ID and public gateway URL. |
| **Tunnels** | A team member runs `mcpzero tunnel start` to bind local MCP servers to an endpoint. Other members call the same URL with their API keys. |
| **API keys** | Generated per user or per integration in [Dashboard → API Keys](/app/api-keys). Keys authenticate client requests at the edge. |
| **Audit visibility** | Team and Enterprise plans retain payloads and provide searchable audit logs so every tool call is traceable. |

## Typical team workflow

1. An admin creates an endpoint in the Dashboard.
2. A developer starts a multiplexed tunnel with `--mcp-auto`, exposing postgres, filesystem, and custom tools.
3. Team members configure Cursor or other MCP clients with the endpoint URL and their API keys.
4. Everyone inspects traffic in [Dashboard → Activity](/app/activity) — tool names, latency, and (on paid plans) full payloads.

## Cross-endpoint composition (Team+)

On **Team** and **Enterprise** plans, teams can aggregate tools and progressive discovery **across multiple endpoints**. This composes several tunnels and environments into a single governed MCP surface for any agent.

See [Semantic aggregation](/docs/gateway/semantic-aggregation/) for details.

## Security considerations

- API keys are scoped per user — rotate or revoke keys without affecting other members.
- Upstream credentials stay on the machine running the tunnel; the gateway never sees them.
- Metadata logging is enabled on all plans; payload retention depends on your plan tier.

## Plan requirements

| Capability | Minimum plan |
|------------|--------------|
| Share endpoints & tunnels | Free |
| Payload cloud storage | Personal |
| Searchable audit logs & team environment sharing | Team |
| Private deployment & DLP | Enterprise |

See [Plans & pricing](/docs/pricing/plans/) for the full comparison.

## Next

- [Security](/docs/gateway/security/)
- [Getting started](/docs/getting-started/overview/)
- [Cursor setup](/docs/cli/cursor/)
