---
title: MCPZERO vs TrueFoundry MCP Gateway
description: Compare TrueFoundry MCP Gateway team auth and credential injection with MCPZERO managed aggregation for local MCP servers.
---

**One-sentence answer:** **TrueFoundry’s MCP Gateway** emphasizes enterprise MCP authentication — centralized OAuth, injected upstream credentials, and team identity for many remote MCP servers; **MCPZERO** focuses on publishing **local** MCP servers via CLI tunnel with semantic aggregation, progressive discovery, and API-key governance.

## Quick comparison

| Dimension | TrueFoundry MCP Gateway | MCPZERO |
|-----------|-------------------------|---------|
| **Primary job** | Team auth & credential hub for MCP | Publish / aggregate / audit local MCP |
| **Credential model** | Gateway holds/injects upstream OAuth & keys | Upstream stays on tunnel host |
| **Local stdio publish** | Not the main story | Core product path |
| **Progressive discovery** | Depends on product surface | Built-in meta server |
| **Best fit** | Orgs unifying many SaaS MCP remotes | Devs exposing laptop/CI tools |

## What TrueFoundry emphasizes

TrueFoundry’s writing on [MCP authentication in Cursor](https://www.truefoundry.com/blog/mcp-authentication-in-cursor-oauth-api-keys-and-secure-configuration) frames the gateway as SSO-like for MCP: developers authenticate once; the gateway maps identity to downstream credentials so tokens do not live in every `mcp.json`.

## What MCPZERO differs

MCPZERO keeps **upstream MCP secrets on the machine running the tunnel** and exposes a **single aggregated endpoint** for Cursor with progressive discovery. API keys authorize clients at the edge; they are not a full OIDC IdP product.

## When to choose TrueFoundry

- Many **already-remote** MCP/SaaS tools need **central OAuth** and per-user token mapping
- Cursor configs must not hold upstream secrets
- Platform team already adopts TrueFoundry

## When to choose MCPZERO

- Problem is **local tools → remote AI clients**
- You want aggregation + progressive discovery out of the box
- Edge API keys + ledger are enough governance for your stage

## Also compare

- [vs Composio](/docs/compare/vs-composio/)
- [vs Kong](/docs/compare/vs-kong-mcp-gateway/)
- [vs MCP security proxies](/docs/compare/vs-mcp-security-proxies/)
- [Comparison hub](/docs/compare/)

## References

- [MCP Authentication in Cursor (TrueFoundry)](https://www.truefoundry.com/blog/mcp-authentication-in-cursor-oauth-api-keys-and-secure-configuration)
- [Evaluating five MCP gateways for production](https://dev.to/sahajmeet_kaur_/what-i-learned-evaluating-five-mcp-gateways-for-production-2clg) — DEV
