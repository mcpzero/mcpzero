---
title: FAQ
description: Frequently asked questions about MCPZERO — secure MCP aggregation, progressive discovery, auth, pricing, and team sharing.
---

**MCPZERO** is a secure MCP aggregation gateway. In one sentence: it publishes, aggregates, and secures local MCP servers behind one zero-trust gateway URL so agents can discover tools on demand instead of loading every schema upfront.

## What is MCPZERO?

MCPZERO is a secure MCP aggregation gateway. It turns local MCP servers into remote HTTPS endpoints that AI clients like Cursor can call, with semantic aggregation, progressive discovery, zero-trust API key auth, and an audit ledger.

## How is MCPZERO different from connecting MCP servers directly?

Direct MCP connections expose every tool schema upfront and often keep credentials on the client machine. MCPZERO tunnels local servers through a zero-trust edge gateway, aggregates many servers behind one URL, and lets agents discover tools on demand via `meta_search` instead of loading the full catalog.

See [MCPZERO vs direct MCP](/docs/compare/vs-direct-mcp/) for a full comparison.

## Is MCPZERO just a tunnel like ngrok?

No. A tunnel-only tool forwards TCP or HTTP traffic. MCPZERO is an MCP-aware gateway: it authenticates API keys at the edge, multiplexes multiple MCP servers behind one endpoint, exposes a meta server for progressive discovery, and records an audit ledger. Tunnels are one transport layer inside that stack.

See [MCPZERO vs tunnel-only tools](/docs/compare/vs-tunnel-only/) and [vs ngrok](/docs/compare/vs-ngrok/).

## Is MCPZERO an alternative to ngrok for MCP?

Yes, when you need an MCP-aware stack rather than only a public URL. ngrok is excellent for HTTP exposure and Traffic Policy; MCPZERO reads `mcp.json`, aggregates many stdio servers, adds progressive discovery, and records an MCP call ledger. See [vs ngrok](/docs/compare/vs-ngrok/) and the [three-way roundup](/docs/compare/mcp-gateways-2026/).

## MCPZERO vs Microsoft MCP Gateway?

Microsoft MCP Gateway is a self-hosted Kubernetes reverse proxy with Entra-oriented identity. MCPZERO is a managed product: CLI tunnel, semantic aggregation, progressive discovery, and API keys without operating AKS. See [vs Microsoft MCP Gateway](/docs/compare/vs-microsoft-mcp-gateway/).

## What is the best MCP gateway for Cursor remote servers?

It depends. For Azure/K8s platform teams, consider Microsoft. For a single HTTP MCP demo, ngrok or Cloudflare Tunnel may suffice. For local `mcp.json` servers you want to share with Cursor via one HTTPS URL, progressive discovery, and audit, start with [MCPZERO](/docs/getting-started/overview/) and the [comparison hub](/docs/compare/).

## What is semantic aggregation?

Semantic aggregation multiplexes one or more backend MCP servers behind a single endpoint root URL. Clients talk to the meta server (`meta_search` and `meta_call_tool`) rather than enumerating every upstream tool at connect time.

## What is progressive discovery?

Progressive discovery means AI clients discover tools on demand. At the endpoint root, `tools/list` returns only `meta_search` and `meta_call_tool`; agents search by intent, then call the matched tool. Server Skill-style profiles are exposed as MCP resources for context before searching.

## Do I need to change my existing mcp.json?

Usually no. The `mcpzero` CLI reads your existing MCP config, starts a tunnel to your endpoint, and keeps upstream credentials on your machine. Point Cursor or other clients at the gateway URL with an API key.

## How does authentication work?

Clients can authenticate with **MCP OAuth 2.1** (browser consent, JWT scoped to endpoints or clusters) or `Authorization: Bearer <mz_live_api_key>`. API keys are created in the Dashboard. OAuth setup for Cursor, Claude Code, and Codex is documented in [MCP OAuth 2.1](/docs/gateway/oauth/). Upstream MCP credentials stay on your machine or in Secret Vault; the gateway never stores them as part of client auth.

## What is an endpoint cluster?

An endpoint cluster (`epc_*`) is a Team+ virtual meta server that aggregates multiple endpoints for cross-endpoint semantic search and progressive discovery from one URL.

## Is there a free plan?

Yes. The Free plan includes one multiplexed tunnel, up to 5 servers and 50 tools total, semantic aggregation and progressive discovery, 48-hour cloud payload storage, and traffic visualization. Paid plans add more tunnels, retention, team sharing, and endpoint clusters.

## Where do I find documentation?

Product docs live at [mcpzero.io/docs](/docs/). Start with [Getting started](/docs/getting-started/overview/), then [CLI install](/docs/cli/install/) and gateway concepts ([semantic aggregation](/docs/gateway/semantic-aggregation/), [progressive discovery](/docs/gateway/progressive-discovery/), [security](/docs/gateway/security/)).

## How do I install the CLI?

On macOS or Linux run:

```bash
curl -fsSL https://mcpzero.io/install-cli.sh | sh
```

Then `mcpzero login` and `mcpzero init`. Homebrew and npm installs are also available — see [Install](/docs/cli/install/).

## Can teams share MCP endpoints?

Yes on Team+. Members call shared endpoints with their own API keys. Tool permissions and audit logs keep access governed. See [Team sharing](/docs/gateway/team-sharing/).

## Does MCPZERO store tool call payloads?

The gateway always records tool name, latency, and status. Cloud payload retention depends on plan: Free 48 hours, Personal 7 days, Team 30 days, Enterprise private BYO storage.
