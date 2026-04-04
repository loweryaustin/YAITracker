# Contributing to YAITracker

Thanks for your interest. This document matches how maintainers work day to day; following it keeps history reviewable and CI predictable.

## Before you start

1. Open or find a **YAIT-*** issue** in the tracker (or ask to have one created) so work has a number for commits and PRs.
2. Base feature work on **`develop`**, not `master`. `master` is for releases and tagged history.

## Branch names

Use Gitflow-style names tied to the issue:

- `feature/YAIT-N-short-description`
- `fix/YAIT-N-short-description`
- `hotfix/YAIT-N-short-description` (urgent production fixes)
- `release/vX.Y.Z` (release preparation only)

## Commits

We use [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>

Optional body paragraphs.

Refs: YAIT-N
```

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

**Scopes:** e.g. `mcp`, `handler`, `store`, `api`, `ci`, `deps` (see `.cursor/rules/git-workflow.mdc`)

Include **`Refs: YAIT-N`** in the commit body when it maps to a tracker issue.

## Local git hooks (recommended)

After cloning, run:

```bash
make hooks
```

That installs `.githooks`: `commit-msg` (conventional first line + reminder about `Refs:`) and `pre-push` (`go vet` + `go test -race`). Hooks can be bypassed with `git commit --no-verify` / `git push --no-verify` or `GITHOOKS_SKIP=1` for push — use sparingly.

## Before opening a PR

Run the full quality suite (required by project guidelines):

```bash
make audit
```

`make audit` runs tests, lint, `go vet`, module checks, and `govulncheck`.

## Pull requests

- Open PRs against **`develop`** unless a maintainer asks otherwise (e.g. hotfix flow).
- Use the PR template checklist.
- PR **title** should follow conventional commits (e.g. `feat(mcp): add export tool`) — CI enforces this.
- Keep changes focused; one logical change per PR when possible.

## CI checks on pull requests

On PRs targeting `master` or `develop`, GitHub Actions runs:

- **Semantic PR title** — the PR title must match conventional commits (e.g. `feat(mcp): add tool` or `fix: handle edge case`).
- **Conventional commit subjects** — each non-merge commit in the PR must use a conventional first line (`feat`, `fix`, `chore`, …).

Workflows: `.github/workflows/ci.yml` (tests, lint, etc.) and `.github/workflows/pr-conventions.yml` (title + commits).

## Merges and history

- Prefer **merge commits** when integrating branches so Gitflow “bubbles” stay visible, unless the branch is noisy and a squash merge was agreed.
- Do **not** force-push to **`master`** or **`develop`** except in exceptional cases coordinated with maintainers.
- This repo may not use GitHub branch protection; discipline is social plus hooks and CI — please follow the same rules anyway.

## License

By contributing, you agree your contributions are licensed under the same terms as the project ([AGPL-3.0](LICENSE)).
