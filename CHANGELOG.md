# Changelog

All notable changes to YAITracker are documented here.
This file is auto-generated from [conventional commits](https://www.conventionalcommits.org/).

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
