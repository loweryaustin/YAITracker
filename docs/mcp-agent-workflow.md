# MCP agent workflow (product)

YAITracker's MCP server is for **automated clients** (IDE agents, scripts using MCP). Normal human use is the **web UI** and **HTTP API**.

## Connection methods

### Sidecar proxy (recommended for IDEs)

The `yaitracker sidecar` command is a stdio-to-HTTP proxy. The IDE launches it as a subprocess and communicates via stdin/stdout JSON-RPC. The Sidecar handles all actor management automatically:

- Registers a process-level MCP actor on startup
- Reads `conversation_id` from the `beforeMCPExecution` hook side-channel to assign per-conversation actors
- Sends heartbeats every 5 minutes to keep actors alive
- Revokes all actors on graceful shutdown or parent process death

Configure in `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "yaitracker": {
      "command": "/path/to/yaitracker",
      "args": ["sidecar"],
      "env": {
        "YAITRACKER_URL": "http://localhost:8080",
        "YAITRACKER_OAUTH_ACCESS_TOKEN": "<from POST /api/v1/auth/token>"
      }
    }
  }
}
```

### HTTP MCP (degraded mode)

Connect directly to the central server's `/mcp` endpoint with static headers.

```json
{
  "mcpServers": {
    "yaitracker": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer <token>",
        "X-Yaitracker-Mcp-Actor-Id": "<id from POST /api/v1/mcp/actors>"
      }
    }
  }
}
```

**Limitation:** When using HTTP transport without the Sidecar, the MCP actor identity is static. All concurrent chats share one actor ID. Deploy the Sidecar for per-conversation isolation and accurate per-task timing.

## MCP actor identity (required)

Agent timers are bound to a **server-issued** MCP actor id. Every agent MCP request must identify its actor:

- **Sidecar:** automatic — the Sidecar injects `X-Yaitracker-Mcp-Actor-Id` on every proxied request
- **HTTP MCP:** manual — the client sends `X-Yaitracker-Mcp-Actor-Id` in the request headers

Register actors via `POST /api/v1/mcp/actors` (bearer-protected). Revoke with `DELETE /api/v1/mcp/actors/{id}`.

There is **no** anonymous agent mode: tools that start or stop timers require a valid actor context.

## Actor lifecycle

1. **Registration:** `POST /api/v1/mcp/actors` with bearer token and optional `label`
2. **Heartbeat:** `POST /api/v1/mcp/actors/{id}/heartbeat` — resets the TTL clock
3. **Expiration:** Actors not heartbeated within **15 minutes** are automatically revoked by the server's cleanup loop; any open agent timers on those actors are stopped
4. **Revocation:** `DELETE /api/v1/mcp/actors/{id}` — immediate manual revocation

The Sidecar handles steps 1-3 automatically. HTTP MCP clients must heartbeat manually or accept that idle actors will be revoked.

## begin_work and complete_work

- **`begin_work`** ensures a work session, moves the issue to `in_progress`, and starts an **agent** timer bound to the current MCP actor
- **`complete_work`** stops the agent timer for **this** MCP actor on that issue, adds a summary comment, and moves the issue to `done`

`complete_work` requires an active agent timer on that issue (i.e. you must have called `begin_work` first). Override with `YAITRACKER_STRICT_AGENT_WORKFLOW=false` on the server (testing only).

## Parallel agents

- **`begin_work` does not stop** agent timers on **other** issues for the same user. You can have one agent timer per issue **per registered MCP actor** across several tickets.
- **Same issue, two agents:** register **two** MCP actors (the Sidecar does this automatically for different conversations). The unique index is `(issue_id, mcp_actor_id)` for open agent timers.

## Work session

There is at most **one** active `work_sessions` row per user. Parallel agents do not fight over the session row.

## Audit

Successful **`begin_work`** / **`complete_work`** append **`activity_log`** rows (`mcp_begin_work`, `mcp_complete_work`) with field **`mcp_actor_id`**.

## Cursor hooks (this repository)

If `.cursor/hooks.json` points at `scripts/cursor-hooks/before-shell.sh` and `before-mcp.sh`:

- **`beforeMCPExecution`** relays `conversation_id` to `.cursor/yait-conversation-id` (read by the Sidecar for per-conversation actors) and manages the work lock for shell gating
- **`beforeShellExecution`** checks the work lock before allowing shell commands

Hook paths are relative to the `hooks.json` file. See `.cursor/hooks.json` for the current configuration.

**Overrides:** `YAIT_HOOK_SKIP=1` in the environment, or an empty file `.cursor/yait-hook-disable`, disables the shell gate (emergencies only).

## Issue labels

Use **`add_issue_label`** to attach labels without the web UI.

## Cursor-specific files

Editor integration (`.cursor/rules`, `.cursor/hooks.json`) applies only to developers using Cursor. It is **not** part of a default YAITracker installation.
