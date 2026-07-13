---
title: Plans & pricing
description: MCPZERO pricing tiers — Free, Personal, Team, and Enterprise.
---

MCPZERO offers the same secure gateway at every tier. You pay for tunnels, aggregation capacity, collaboration, audit retention, and advanced security controls as you grow.

## Compare plans

| | Free | Personal | Team | Enterprise |
|---|:---:|:---:|:---:|:---:|
| **Price** | $0 forever | $5 / mo · $50 / yr | $29 / mo · $290 / yr | Custom |
| **Tunnels** | 1 | 2 | 10 | Unlimited |
| **Servers per tunnel** | 5 (1 tunnel) | 5 | 10 | Custom |
| **Tools (aggregated)** | 50 total | 50 / tunnel | 200 / tunnel | Custom |
| **Team members** | 1 | 1 | 5 | Custom |
| **Semantic aggregation** (within one endpoint) | Yes | Yes | Yes | Yes |
| **Progressive discovery** (within one endpoint) | Yes | Yes | Yes | Yes |
| **Cross-endpoint aggregation & discovery** | — | — | Yes ([endpoint clusters](/docs/gateway/endpoint-clusters/)) | Yes |
| **Traffic visualization / Overview SLO** | Yes | Yes | Yes | Yes |
| **Cloud payload storage** | 48 hours | 7 days | 30 days | Private BYO |
| **Searchable audit logs** | Yes | Yes | Yes | Yes |
| **Exportable audit logs** | — | Yes (≤1k/export) | Yes (≤5k/export) | Yes |
| **Webhook alerts** | — | Yes (account) | Yes (account + team) | Yes |
| **Rate limit** | 30 / min | 30 / min | 60 / min per endpoint | Custom |
| **Semantic WAF** *(Roadmap)* | — | — | — | Yes |
| **Tool Hijacking Defense** *(Roadmap)* | — | — | — | Yes |
| **DLP & compliance controls** *(Roadmap)* | — | — | — | Yes |
| **Private deployment / hybrid cloud** | — | — | — | Yes |
| **Dedicated support & SLA** | — | — | — | Yes |

> Enterprise security features marked *Roadmap* are on the product roadmap. Contact [sales@mcpzero.io](mailto:sales@mcpzero.io) for availability and deployment options.

## Plan details

### Free

For developers exploring MCP aggregation. One tunnel with up to 5 servers and 50 tools total across the account. Semantic aggregation and progressive discovery at the endpoint root. **48-hour** cloud payload storage, traffic visualization / Overview SLO, and searchable Activity. Export and webhook alerts require Personal or Team.

### Personal

For individuals who need longer payload history and actionable alerts. Two tunnels, each supporting up to 5 servers and 50 aggregated tools. Includes **7-day** cloud storage for request/response payloads, NDJSON audit export (capped), and account webhook alerts. Does not include cross-endpoint aggregation or team collaboration.

### Team

For small teams sharing MCP capabilities. Ten tunnels, each supporting up to 10 servers and 200 aggregated tools. Up to **5 team members**. [Endpoint clusters](/docs/gateway/endpoint-clusters/) (`epc_*`) for cross-endpoint aggregation and discovery, 30-day cloud storage, higher audit export caps, and team-scoped webhook alerts. Rate limit of 60 requests per minute per endpoint.

### Enterprise

For organizations with compliance, scale, or deployment requirements. Unlimited tunnels with custom server, tool, and team-member limits. Private cloud storage (S3 / R2 / OSS), Semantic WAF, Tool Hijacking Defense, and DLP *(Roadmap)*, private or hybrid-cloud deployment, custom rate limits, and dedicated support with SLA.

## What's included at every tier

- Zero-trust edge authentication with API keys
- Encrypted WebSocket tunnel from local MCP servers to the gateway
- Metadata call ledger (tool name, latency, status) with searchable Activity
- Traffic visualization / Overview SLO (24h / 7d / 30d)
- [Semantic aggregation](/docs/gateway/semantic-aggregation/) and [progressive discovery](/docs/gateway/progressive-discovery/) at the endpoint root

Personal and Team add NDJSON audit export and webhook alerts (Team also adds higher export caps and team-scoped alerts).

## Get started

[Create a free account](https://mcpzero.io/app/register) or see the [landing page pricing section](https://mcpzero.io/#pricing) for the latest details.

For Enterprise inquiries, contact [sales@mcpzero.io](mailto:sales@mcpzero.io).
