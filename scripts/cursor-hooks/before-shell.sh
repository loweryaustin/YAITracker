#!/usr/bin/env bash
# Cursor beforeShellExecution hook (beta). Reads JSON from stdin; prints allow/deny JSON.
# Git discipline uses .githooks (commit-msg, pre-push). This hook does not block commands.
set -euo pipefail
cat >/dev/null || true
printf '%s\n' '{"permission":"allow"}'
