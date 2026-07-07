---
title: Init (onboarding)
description: Interactive first-time setup — endpoint, API key, Cursor config, and tunnel.
---

`mcpzero init` is the fastest way to go from a fresh install to a working Cursor
connection. In an interactive terminal it walks you through four steps:

1. **Endpoint** — pick an existing endpoint or create one
2. **API key** — create a new key or paste one you saved earlier
3. **Cursor** — write `~/.cursor/mcp.json` (or project `.cursor/mcp.json`)
4. **Tunnel** — optionally start `mcpzero tunnel start --mcp-auto` (foreground or background)

If you are not logged in, `init` runs `mcpzero login` first.

## Quick start

```bash
mcpzero login    # skip if already logged in
mcpzero init
```

Follow the prompts. When the wizard finishes you have:

- A gateway URL like `https://gw.mcpzero.io/v1/ep_abc123`
- Cursor configured with `Authorization: Bearer mz_live_…`
- (Optional) a tunnel running with your local MCP servers

## Interactive flow

### Step 1 — Endpoint

- Lists endpoints you already own (name, ID, scope, status).
- If your plan still has quota, offers **Create a new endpoint**.
- On **Free** (1 personal endpoint), when the limit is reached the wizard only
  lets you pick an existing endpoint — it does not offer create.

### Step 2 — API key

- Shows hints for keys already scoped to this endpoint (e.g. `mz_live_ab12...xy34`).
  Raw keys are **only shown at creation** — the CLI cannot retrieve them later.
- **Create a new API key** (recommended) — prints the raw key once at the end.
- **Paste a saved key** — use a `mz_live_…` value from your password manager.

### Step 3 — Cursor

Choose where to write the MCPZERO HTTP entry:

| Option | Target |
|--------|--------|
| `0` | Skip — don't write Cursor `mcp.json` |
| `1` | Global — `~/.cursor/mcp.json` (default) |
| `2` | Project — `.cursor/mcp.json` in the current directory |

Default server name: `mcpzero`. URL is the **endpoint root** (`/v1/ep_…`) for semantic aggregation and progressive discovery.

### Step 4 — Tunnel

- **Start tunnel now?** — runs `mcpzero tunnel start --endpoint … --mcp-auto`
  and auto-discovers MCP servers from your installed agents.
- **Run in background?** — frees the terminal (`-d`). See
  [Background tunnels](#background-tunnels) below.

Skip tunnel setup with `--no-tunnel` if you only want endpoint + Cursor config.

## Non-interactive mode

For scripts, CI, or pipes, pass `-y` / `--yes`:

```bash
mcpzero init -y
```

Defaults:

- Reuse the newest personal endpoint when one exists; otherwise create `default`.
- Create a new scoped API key.
- Write global `~/.cursor/mcp.json`.
- Print the tunnel start command (does not start the tunnel unless you ran the
  interactive wizard without `-y`).

### Useful flags

| Flag | Description |
|------|-------------|
| `-y`, `--yes` | Non-interactive mode |
| `--endpoint ep_…` | Reuse a specific endpoint |
| `--new` | Force create a new endpoint (fails when plan limit is reached) |
| `--api-key mz_live_…` | Use an existing key (skip key creation) |
| `--endpoint-name name` | Name for a new endpoint (`-y` only) |
| `--name mcpzero` | Cursor `mcpServers` entry name |
| `--project` | Write `.cursor/mcp.json` in the current directory |
| `--no-cursor` | Skip Cursor config |
| `--no-tunnel` | Skip tunnel prompt / command |
| `--team-id team_…` | Create under a team (when creating) |

Examples:

```bash
# Reuse endpoint + saved key, configure Cursor only
mcpzero init -y --endpoint ep_abc123 --api-key mz_live_… --no-tunnel

# Full non-interactive setup with a new key
mcpzero init -y --endpoint-name "my tools"
```

## Background tunnels

When you choose **background** during `init`, the CLI prints:

```text
mcpzero tunnel attach <id>   # watch logs in foreground (Ctrl-C detaches)
mcpzero tunnel logs <id> -f
mcpzero tunnel stop <id>

# Run in the foreground instead (Ctrl-C stops the tunnel):
mcpzero tunnel stop <id>
mcpzero tunnel start --endpoint ep_… --mcp-auto
```

- **attach** / **logs -f** — foreground log view; Ctrl-C only detaches, the tunnel
  keeps running.
- **stop** then **tunnel start** without `-d` — true foreground mode where Ctrl-C
  stops the tunnel.

## Related commands

| Command | When to use |
|---------|-------------|
| `mcpzero cursor add` | Add or update Cursor config without re-running the full wizard |
| `mcpzero tunnel start` | Start or restart a tunnel later |
| `mcpzero whoami` | Confirm logged-in account |

## Next

- [Cursor setup](/docs/cli/cursor/) — manual configuration and curl tests
- [Tunnel](/docs/cli/tunnel/) — flags, multiplexing, background mode
- [Troubleshooting](/docs/cli/troubleshooting/)
