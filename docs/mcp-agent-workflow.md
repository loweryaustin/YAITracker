# MCP agent workflow (product)

YAITracker’s MCP server is for **automated clients** (IDE agents, scripts using MCP). Normal human use is the **web UI** and **HTTP API**.

## begin_work and complete_work

- **`begin_work`** ensures a work session, moves the issue to `in_progress`, and starts an **agent** timer on the issue.
- **`complete_work`** stops timers on that issue, adds an optional summary comment, and moves the issue to `done`.

With default server settings, **`complete_work` requires an active agent timer on that issue** (i.e. you must have called `begin_work`, or with strict workflow enabled, `start_timer` follows the same preparation as `begin_work`). Otherwise the tool returns an error asking you to call `begin_work` first.

## Operator override

Set environment variable **`YAITRACKER_STRICT_AGENT_WORKFLOW=false`** (or `0`, `no`, `off`) on the server process to disable the active-timer check for `complete_work` and to allow legacy `start_timer` behavior. Use only for testing, migration, or emergencies — not typical production use.

## Parallel agents (one human, multiple issues)

- **`begin_work` does not stop** agent timers on **other** issues for the same user. You can have one agent timer per issue across several tickets.
- **One agent per ticket:** do not run two agents on the **same** issue at once; split work into subtasks instead.
- **Work session:** there is still at most **one** active `work_sessions` row per user. The default policy (**1a**) is not to overwrite that session’s description on every `begin_work` when a session already exists, so parallel agents do not fight over the same row.

## Cursor shell hooks (this repository)

If `.cursor/hooks.json` points at `scripts/cursor-hooks/before-shell.sh` and `before-mcp.sh`:

- **`begin_work(project_key: "YAIT", number: N)`** creates `.cursor/yait-work-lock` (gitignored) so **agent shell commands** are allowed afterward.
- **`complete_work`** for that same issue removes the lock.
- **Overrides:** `YAIT_HOOK_SKIP=1` in the environment, or an empty file `.cursor/yait-hook-disable`, disables the shell gate (emergencies only).
- A small **read-only allowlist** (e.g. `git status`, `git diff`, `go version`) works without a lock so you can inspect the repo before `begin_work`.

Hooks only affect **Cursor**; they are not part of a default server install.

## Cursor-specific files

Editor integration (e.g. `.cursor/rules`, `.cursor/hooks.json`) applies only to developers using those tools in a checkout. It is **not** part of a default YAITracker installation.
