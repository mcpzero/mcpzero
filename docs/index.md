---
title: MCPZERO Docs
description: Tunnel, authenticate, and observe MCP servers from your laptop to production clients.
template: splash
hero:
  tagline: MCP routing infrastructure for developers shipping local MCP servers to Cursor and other AI clients.
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

- **Tunnel local MCP** — Run `mcpzero tunnel start` to expose a stdio MCP server through the MCPZERO gateway.
- **Protect with API keys** — Buyers call your endpoint with `Authorization: Bearer <mz_live_api_key>`. Keys are managed in the Dashboard.
- **Observe every tool call** — The gateway records tool name, latency, payloads, and auth failures in the call ledger.
- **Tunnel from your code** — Use `mcpzero-sdk` to expose your MCP server through the tunnel directly from code — TypeScript, Python, Go, and Rust.
