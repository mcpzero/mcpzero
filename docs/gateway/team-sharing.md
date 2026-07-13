---
title: Team sharing
description: Share MCP endpoints, tunnels, and audit visibility across your team.
---

MCPZERO lets teams **share MCP capabilities** — expose local tools through governed endpoints that every member can call, inspect, and audit from a central dashboard.

## What teams share

| Resource | How it works |
|----------|--------------|
| **Endpoints** | Created in [Dashboard → Endpoints](/app/endpoints). Each endpoint has a unique ID and public gateway URL. |
| **Tunnels** | A team member runs `mcpzero tunnel start` to bind local MCP servers to an endpoint. Other members call the same URL with their API keys. |
| **API keys** | Generated per user in [Dashboard → API Keys](/app/api-keys). Keys authenticate client requests at the edge. |
| **Audit visibility** | Every plan has a searchable Activity ledger. Payload retention and NDJSON export scale by plan; Team adds shared team-pool visibility and higher export caps. |

## Typical team workflow

1. A **Team plan** owner creates a team in the Dashboard and invites members (up to 5 on Team).
2. An admin or developer creates an endpoint in [Dashboard → Endpoints](/app/endpoints).
3. A developer starts a multiplexed tunnel: `mcpzero tunnel start --endpoint ep_… --mcp-auto`
4. Each member configures Cursor at the **endpoint root** URL with their own API key.
5. Everyone inspects traffic in [Dashboard → Activity](/app/activity) — tool names, latency, and (on paid plans) payloads.
6. On **Personal+**, use **Export NDJSON** on Activity to download matching rows as a zip (Personal ≤1,000 rows; Team ≤5,000; download link expires in 24 hours). Configure account webhook alerts under [Overview](/app/overview) (Team owners can also set team-scoped alerts under [Team](/app/team)).

Members inherit the team owner's plan tier for gateway limits (cross-endpoint features, rate limits, etc.).

## Cross-endpoint composition (Team+)

On **Team** and **Enterprise** plans, teams can aggregate tools and progressive discovery **across multiple endpoints** using **endpoint clusters** (`epc_*`). Create a cluster in [Dashboard → Team → Clusters](/app/team/clusters), add member endpoints owned by team members, and point agents at the cluster root URL.

See [Endpoint clusters](/docs/gateway/endpoint-clusters/) for the full workflow, meta-tool differences (`endpoint_id` in `meta_call_tool`), and Cursor setup.

## Security considerations

- API keys are scoped per user — rotate or revoke keys without affecting other members.
- Upstream credentials stay on the machine running the tunnel; the gateway never sees them.
- Metadata logging is enabled on all plans; payload retention depends on your plan tier.

## Plan requirements

| Capability | Minimum plan |
|------------|--------------|
| Share endpoints & tunnels | Free |
| Searchable Activity / Overview SLO | Free |
| Up to 5 servers / 50 tools | Free & Personal |
| Payload cloud storage (7d+) | Personal |
| NDJSON audit export | Personal |
| Account webhook alerts | Personal |
| Up to 10 servers / 200 tools per tunnel, 5 members, cross-endpoint, team alerts | Team |
| Private deployment & DLP *(Roadmap)* | Enterprise |

See [Plans & pricing](/docs/pricing/plans/) for the full comparison.

## Next

- [Security](/docs/gateway/security/)
- [Getting started](/docs/getting-started/overview/)
- [Cursor setup](/docs/cli/cursor/)
