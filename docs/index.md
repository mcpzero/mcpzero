---
title: MCPZERO Docs
description: Publish, aggregate, and secure local MCP servers — tunnel, aggregate, discover, and audit.
template: splash
hero:
  tagline: Publish, aggregate, and secure your local MCP servers — semantic aggregation, progressive discovery, zero-trust security, and audit visibility.
  actions:
    - text: Get started
      link: /docs/getting-started/overview/
      icon: right-arrow
      variant: primary
    - text: Install CLI
      link: /docs/cli/install/
      icon: external
      variant: minimal
    - text: Open Dashboard
      link: /app/register
      icon: right-arrow
      variant: minimal
---

## What you can do

- **Aggregate MCP servers** — Multiplex local stdio servers through one endpoint. Connect at the **endpoint root** for semantic aggregation and progressive discovery.
- **Compose across endpoints** — On Team+, create **endpoint clusters** (`epc_*`) to search and call tools across several tunnels from one URL.
- **Share with your team** — Expose governed MCP endpoints that members can call with their own API keys from the Dashboard.
- **Protect with zero-trust auth** — Clients authenticate with `Authorization: Bearer <mz_live_api_key>`. Upstream credentials never leave your machine.
- **Observe every tool call** — The gateway records tool name, latency, and status by default. Payload retention depends on your plan.

## Compare & use cases

- [FAQ](/docs/faq/) — common questions about MCPZERO, auth, pricing, and discovery
- [MCP gateway comparison](/docs/compare/) — matrix vs ngrok, Microsoft, Docker, Kong, and more
- [ngrok vs Microsoft MCP Gateway vs MCPZERO](/docs/compare/mcp-gateways-2026/) — three-way 2026 roundup
- [vs direct MCP](/docs/compare/vs-direct-mcp/) — when to use a gateway instead of client-local MCP
- [vs tunnel-only tools](/docs/compare/vs-tunnel-only/) — MCP-aware gateway vs generic tunnels
- [Multiple servers in Cursor](/docs/use-cases/multi-server-cursor/)
- [Team-shared endpoint](/docs/use-cases/team-shared-endpoint/)
- [Expose a SQLite database](/docs/use-cases/sqlite-database/)
