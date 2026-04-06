# Changelog

All notable changes to YAITracker are documented here.
This file is auto-generated from [conventional commits](https://www.conventionalcommits.org/).


## Bug Fixes

- Resolve all 331 pre-existing golangci-lint errors (2002706)
- Store CSRF token in request context for first-load correctness (e9b27fa)
- Render markdown in issue descriptions and comment bodies (b20d6e5)
- Remove hard limit cap that hid board issues beyond 25 rows (4316c3f)

## Features

- Auto-refresh expired OAuth tokens in sidecar proxy (42de74c)


## Features

- Add sidecar stdio-to-HTTP proxy with per-conversation actor isolation (21888a6)


## Bug Fixes

- Locate go when PATH omits install dirs (2f33c9a)

## Documentation

- Align workflow rules with canonical git and MCP flow (0b24bcb)

## Features

- Attribute agent timers with MCP actor id (cad0cbf)

## Miscellaneous

- Record cursor rules alignment under YAIT-32 (d22e0e2)

## Testing

- Add NewTestStoreFile for concurrent store access (96690f1)


## Features

- Add add_issue_label tool (2f7b22a)
- Cursor hooks require begin_work; test parallel agent timers (39d3b2c)


## Bug Fixes

- Require VERSION tag to be on master before deploy (2cbb247)
- Expose issue parent and child links in MCP tools (0903330)
- Use typed gt helpers on project analytics page (04312c2)

## Documentation

- Add CONTRIBUTING.md and pull request template (7b183f2)
- Align Gitflow rules with master branch (edabcc0)

## Features

- Enforce agent timer before complete_work (48cc52a)
- Add shared git hooks and make hooks target (37ee3b9)

## Miscellaneous

- Pin govulncheck version and sync notes with Makefile (7d09997)
- Run lint and vulncheck via pinned go run (82fca89)

## Ci

- Add PR title and commit subject checks (a4e8aee)

## Style

- Gofmt after YAIT-17 merge (e3f079a)


## Bug Fixes

- Add GHCR docker login to release workflow (e3edb80)
- Add GoReleaser Dockerfile and fix docker image path (3a11869)
- Fix release changelog and skip lint on master (6e1aad6)
- Set lint to only-new-issues until YAIT-21 cleans up backlog (ff3cc4b)
- Migrate golangci-lint config to v2 and bump Go to 1.25.8 (07e15fb)
- Bump Go to 1.25.7 and golangci-lint to v2.11.4 (f8d7857)
- Unblock CI and release pipeline (3551bdb)


## Documentation

- Rewrite README for public audience with AI-native positioning (3e23938)

## Features

- Add comprehensive structured data and GEO-optimized robots.txt (54a66b2)
- Add SEO/GEO polish, animated terminal, favicon, OG image, FAQ (e5eb425)
- Add app screenshots and redesign landing page for YAIT-16 (22bbe86)
- Add GitHub Pages landing page at yaitracker.com (290d7e6)


## Bug Fixes

- Improve timer UI readability and restrict MCP to agent-only (f4faeef)


## Bug Fixes

- Stop previous agent timers in begin_work and update session description (3b123d9)


## Features

- Audit repo for public release and add AGPLv3 license (0faedac)


## Features

- Add standardized deployment with backup, health check, and rollback (d5d79aa)


## Features

- Redesign time tracking UI with session banner, time hub, and enhanced views (557f74e)


## Features

- Redesign MCP transport, auth, and workflow tools (2766fb3)


## Bug Fixes

- Add Tailwind CSS build pipeline to fix unstyled web UI (35317ad)


## Bug Fixes

- Add CSP nonces and unsafe-eval for Alpine.js compatibility (d67b20a)


## Features

- Add delete_project tool with cascading delete support (556c623)


## Features

- Replace manual time logging with real-time start/stop timers (e3ba183)


## Miscellaneous

- Initial project setup with full development workflow (ddf23d9)


