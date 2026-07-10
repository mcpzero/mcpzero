---
title: Watch
description: Tail cloud Activity from the terminal with mcpzero watch.
---

`mcpzero watch` polls your cloud Activity feed and prints new MCP calls as they
appear — with status, latency, and a Dashboard deep link for each trace.

## Usage

```bash
mcpzero watch
mcpzero watch --endpoint ep_abc123
mcpzero watch --status error --interval 1s
mcpzero watch --search meta_search --once
```

| Flag | Description |
|------|-------------|
| `--endpoint` | Filter by endpoint ID |
| `--status` | Filter by status (`success`, `error`, `timeout`, `auth_denied`) |
| `--search` | Free-text search (tool, IP, server, status, trace id) |
| `--interval` | Poll interval (default `2s`) |
| `--once` | Fetch recent rows once and exit |
| `--limit` | Max rows per poll (default `20`) |

Requires `mcpzero login`. Uses `GET /app/api/cli/activity` with your CLI refresh
token. Team members also see Activity for **team-pool endpoints** they can access.

## Example

```text
watching activity on https://dev.mcpzero.io (Ctrl-C to stop)
2026-07-10 09:01:02  success       list_directory @local-fs-tmp 12ms  ep_abc
  → https://dev.mcpzero.io/app/activity/tr_…
2026-07-10 09:01:05  error         meta_search 840ms  ep_abc
  → https://dev.mcpzero.io/app/activity/tr_…
```

## Next

- [Doctor](/docs/cli/doctor/)
- [Troubleshooting](/docs/cli/troubleshooting/)
