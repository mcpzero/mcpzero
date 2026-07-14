---
title: MCPZERO vs MCP security proxies
description: Compare MCPZERO with MCP Zero-Trust Proxy, mcpgate, and PolicyLayer — self-hosted OAuth/RBAC/policy vs managed aggregation gateway.
---

**One-sentence answer:** **MCP security proxies** (MCP Zero-Trust Proxy, mcpgate, PolicyLayer, and similar) add OAuth, RBAC, deny-by-default, or grant policies in front of MCP servers you already run; **MCPZERO** is a managed aggregation gateway that also publishes local servers remotely — different product layer, sometimes complementary.

> **Name note:** [MCP Zero-Trust Proxy](https://mcpzerotrust.dev/) is an **unrelated open-source project**. It is **not** MCPZERO (`mcpzero.io`).

## Quick comparison

| Dimension | Security / policy proxies | MCPZERO |
|-----------|---------------------------|---------|
| **Primary job** | AuthZ, RBAC, approvals, audit sinks | Publish + aggregate + govern MCP |
| **Deployment** | Self-host on your infra | Managed edge + CLI tunnel |
| **Publishing local servers** | You still provide reachability | Built-in outbound tunnel |
| **Progressive discovery** | Usually pass-through | Built-in meta server |
| **Air-gap / data locality** | Traffic stays on your network | MCP traffic via managed gateway |
| **Best fit** | Compliance, deny-by-default | Remote Cursor without DIY stack |

## What these projects are

- **[MCP Zero-Trust Proxy](https://mcpzerotrust.dev/)** ([GitHub](https://github.com/keith-aykira/mcp-zero-trust-proxy)) — OAuth 2.1 PKCE, tool-level RBAC, rate limits, SIEM sinks in front of existing MCP servers.
- **[mcpgate](https://github.com/maksym-mishchenko/mcpgate)** — deny-by-default policy firewall with SQLite audit and optional human approval.
- **[PolicyLayer](https://policylayer.com/integrations/cursor)** — policy/grant gateway patterns for Cursor and remote MCP URLs.

They answer: “How do I **secure** MCP I already expose?” MCPZERO answers: “How do I **publish, aggregate, and secure** local MCP for AI clients as a product?”

## What MCPZERO differs

MCPZERO includes zero-trust **API keys**, tool permissions, and a ledger, but it is also the **transport and aggregation** layer (tunnel + meta server). Pure security proxies generally assume the network path already exists.

## When to choose a security proxy

- Traffic and logs must **never leave** your VPC / laptop path
- You need **OIDC enterprise IdP**, HMAC audit chains, or interactive **ASK** approval
- You already have remote MCP URLs and only need a policy wrapper

## When to choose MCPZERO

- You need end-to-end **publish → auth → aggregate → audit** without assembling proxy + tunnel + HTTP wrap
- Progressive discovery and Team endpoint clusters are required
- Managed Free/Personal/Team plans are acceptable

**Complementary:** some teams tunnel with MCPZERO for publication and add stricter local proxies for specific high-risk tools — evaluate per threat model.

## Also compare

- [vs Kong](/docs/compare/vs-kong-mcp-gateway/)
- [vs TrueFoundry](/docs/compare/vs-truefoundry/)
- [Security docs](/docs/gateway/security/)
- [Comparison hub](/docs/compare/)

## References

- [MCP Zero-Trust Proxy](https://mcpzerotrust.dev/)
- [keith-aykira/mcp-zero-trust-proxy](https://github.com/keith-aykira/mcp-zero-trust-proxy)
- [maksym-mishchenko/mcpgate](https://github.com/maksym-mishchenko/mcpgate)
- [PolicyLayer × Cursor](https://policylayer.com/integrations/cursor)
- CSA [Agentic MCP Security Best Practices](https://labs.cloudsecurityalliance.org/agentic/agentic-mcp-security-best-practices-v1/)
