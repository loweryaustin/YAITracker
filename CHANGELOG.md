# Changelog

All notable changes to YAITracker are documented here.
This file is auto-generated from [conventional commits](https://www.conventionalcommits.org/).

## [v0.7.0] - 2026-04-04

### Features

- Standardize deployment with backup, health check, and rollback (d5d79aa)

  Add deploy/deploy.sh with pre-flight checks, database backup, compose deploy,
  health check, data verification, cleanup, and automatic rollback on failure.
  All host-specific values live in deploy/.env (gitignored). New Makefile
  targets: deploy, deploy/backup, deploy/rollback. New Cursor rule for
  deployment guidance.

## [v0.6.0] - 2026-04-04

### Features

- Redesign time tracking UI with session banner, time hub, and enhanced views (557f74e)

  Replace tiny header timer widget with a persistent session banner showing
  session duration, human/agent timer status, and clock-in/out controls.
  Add comprehensive `/time` hub page with active timers panel, daily summary
  stats, session history, and improved weekly timesheet with human/agent actor
  type split. Enhance issue detail page with time budget progress bar,
  human/agent breakdown, and actor badges. Add today's time summary card to
  dashboard. New store methods: `ListRecentWorkSessions`,
  `GetSessionUtilization`, `GetDailySummary`, `GetActiveTimersWithIssues`.

## [v0.5.0] - 2026-04-03

### Features

- Redesign MCP transport, auth, and workflow tools (2766fb3)

  Switch from SSE to Streamable HTTP transport at `/mcp`. Add bearer token
  authentication for caller identity. Introduce `begin_work` and `complete_work`
  compound tools that reduce per-task ceremony from 7-8 calls to 1-2. Add
  `delete_issue` tool. Wire `start_timer` description to database. Add
  `actor_type` filter to `stop_timer`. Make `list_issues` concise by default.
  Simplify workflow and code-review rules.

## [v0.4.0] - 2026-04-03

### Bug Fixes

- Add Tailwind CSS build pipeline to fix completely unstyled web UI (35317ad)

  Tailwind utility classes were used across all templates but Tailwind CSS
  was never included. Added Tailwind v4 standalone CLI build, `input.css`
  with `@source` directives for Go handler templates, Makefile `css` targets,
  and Dockerfile integration. Removed conflicting dark-theme custom CSS.

## [v0.3.1] - 2026-04-03

### Bug Fixes

- Fix CSP blocking Alpine.js and inline scripts in web UI (d67b20a)

  Generate per-request cryptographic nonces for inline `<script>` and
  `<style>` tags. Add `'unsafe-eval'` to `script-src` for Alpine.js
  expression evaluation. All interactive UI elements (dropdowns, board
  drag-and-drop, toasts, sidebar) now work correctly.

## [v0.3.0] - 2026-04-03

### Features

- Add `delete_project` MCP tool with cascading delete support (556c623)

  Permanently deletes a project and all associated data (issues, comments,
  time entries, labels, tags, members) via SQLite ON DELETE CASCADE.
  Requires `confirm=true` safety guard. Also fixes unchecked errors in
  REST API and web UI delete handlers.

## [v0.2.0] - 2026-04-03

### Features

- Replace manual time logging with real-time start/stop timers (e3ba183)

  New MCP tools: `start_session`, `end_session`, `start_timer`, `stop_timer`,
  `get_session_status`. Removed `log_time`. Work sessions track human clock-in/out.
  Agent timers can run concurrently on different issues. Orphaned timer cleanup.

## [v0.1.0] - 2026-04-03

### Miscellaneous

- Initial project setup with full development workflow (ddf23d9)

  Cursor rules, golangci-lint, Makefile, test infrastructure, GitHub Actions CI/CD,
  GoReleaser + git-cliff config, SEMVER tooling, version injection.
