# Changelog

All notable changes to YAITracker are documented here.
This file is auto-generated from [conventional commits](https://www.conventionalcommits.org/).

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
