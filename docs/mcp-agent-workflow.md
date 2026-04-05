# MCP agent workflow (product)

YAITracker’s MCP server is for **automated clients** (IDE agents, scripts using MCP). Normal human use is the **web UI** and **HTTP API**.

## begin_work and complete_work

- **`begin_work`** ensures a work session, moves the issue to `in_progress`, and starts an **agent** timer on the issue.
- **`complete_work`** stops timers on that issue, adds an optional summary comment, and moves the issue to `done`.

With default server settings, **`complete_work` requires an active agent timer on that issue** (i.e. you must have called `begin_work`, or with strict workflow enabled, `start_timer` follows the same preparation as `begin_work`). Otherwise the tool returns an error asking you to call `begin_work` first.

## Operator override

Set environment variable **`YAITRACKER_STRICT_AGENT_WORKFLOW=false`** (or `0`, `no`, `off`) on the server process to disable the active-timer check for `complete_work` and to allow legacy `start_timer` behavior. Use only for testing, migration, or emergencies — not typical production use.

## Cursor-specific files

Editor integration (e.g. `.cursor/rules`, `.cursor/hooks.json`) applies only to developers using those tools in a checkout. It is **not** part of a default YAITracker installation.
