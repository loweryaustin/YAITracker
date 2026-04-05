# Changelog

All notable changes to YAITracker are documented here.
This file is auto-generated from [conventional commits](https://www.conventionalcommits.org/).

## [v0.9.1] - 2026-04-04

## [v0.9.2] - 2026-04-05

### Bug Fixes

- Expose issue parent and child links in MCP tools (0903330)

  Refs: YAIT-26

- Use typed gt helpers on project analytics page (04312c2)

### Features

- Enforce agent timer before complete_work (48cc52a)

- Add shared git hooks and `make hooks` target (37ee3b9)

### Continuous integration

- Add PR title and commit subject checks (a4e8aee)

### Documentation

- Add CONTRIBUTING.md and pull request template (7b183f2)

- Align Gitflow rules with master branch (edabcc0)

### Miscellaneous

- Pin govulncheck version and sync notes with Makefile (7d09997)

- Run lint and vulncheck via pinned go run (82fca89)

### Style

- Gofmt after YAIT-17 merge (e3f079a)


### Bug Fixes

- Unblock CI and release pipeline (3551bdb)

  Anchor `.gitignore` `/yaitracker` to root so `cmd/yaitracker/` is not
  ignored; commit `cmd/yaitracker/main.go` which was silently excluded.
  Update CI branch triggers from `main` to `master`. Upgrade
  golangci-lint-action to v7 with golangci-lint v2.11.4 and migrate
  config from `linters-settings` to `linters.settings` (v2 schema).
  Bump Go from 1.25.0 to 1.25.8 to resolve all known stdlib
  vulnerabilities. Set lint `only-new-issues: true` pending YAIT-21
  cleanup of 360 pre-existing errors.

## [v0.9.0] - 2026-04-04

### Features

- Add GitHub Pages landing page at yaitracker.com (290d7e6)

  Static landing page in `docs/` with hero section, feature overview, MCP
  integration showcase, and get started instructions. Custom domain via
  CNAME file. Tailwind CSS via CDN.

  Refs: YAIT-12

- Add app screenshots and redesign landing page (22bbe86)

  Captured 5 browser screenshots (dashboard, board, time tracking, issue
  detail, analytics) via Playwright. Rebuilt landing page with browser-frame
  hero mockup, alternating image/text feature sections, and styled MCP
  integration showcase.

  Refs: YAIT-16

- Add SEO/GEO polish, animated terminal, favicon, OG image, FAQ (e5eb425)

  Animated MCP terminal with IntersectionObserver typing effect. OG social
  preview image and favicon. JSON-LD structured data (SoftwareApplication +
  FAQPage). FAQ section with expandable accordions. Scroll-triggered fade-in
  animations. Gradient mesh backgrounds and section dividers. Semantic HTML
  with ARIA labels.

  Refs: YAIT-16, YAIT-19

- Add comprehensive structured data and GEO-optimized robots.txt (54a66b2)

  WebSite, Person, SoftwareSourceCode, and HowTo JSON-LD schemas.
  Cross-referenced entities via @id URIs. robots.txt explicitly allows
  GPTBot, ClaudeBot, PerplexityBot, Google-Extended, and cohere-ai crawlers.
  sitemap.xml for search indexing.

  Refs: YAIT-19

### Documentation

- Rewrite README for public audience with AI-native positioning (3e23938)

  Updated introductory text, tagline, and "Why YAITracker?" section to
  emphasize AI-era productivity, multi-agent orchestration, and automation.
  Added SEO keywords, competitor mentions (Jira, Linear), and AI tool
  compatibility (Cursor, Copilot, Claude). Contributing section and alpha
  software notice.

  Refs: YAIT-11

## [v0.8.2] - 2026-04-04

### Bug Fixes

- Improve active timer UI readability with full issue key and title (f4faeef)

  Redesign active timer cards on Time page to a two-line layout showing the
  full issue key and elapsed time on top, with the issue title below. Session
  banner now shows individual agent issue keys (e.g. YAIT-14) instead of a
  generic count. Remove `start_session` and `end_session` from MCP tools;
  human time tracking is now UI-only. Force `start_timer` to agent actor type.

## [v0.8.1] - 2026-04-04

### Bug Fixes

- Stop previous agent timers in begin_work and update session description (3b123d9)

  `begin_work` now stops any running agent timers before starting a new
  one, preventing time leaking on old issues. Reusing an existing session
  updates its description to reflect the current issue. Timer description
  is set to the issue title for better UI display.

## [v0.8.0] - 2026-04-04

### Features

- Audit repository for public release and add AGPLv3 license (0faedac)

  Scanned working tree and full git history for secrets, private IPs,
  hostnames, and personal filesystem paths — all clean. Added AGPLv3
  LICENSE file. Updated README with correct GitHub clone URL, current
  MCP tools table (added delete_project, delete_issue, start_session,
  end_session, start_timer, stop_timer, get_session_status, begin_work,
  complete_work), and accurate feature/endpoint descriptions.

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
