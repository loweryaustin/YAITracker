#!/usr/bin/env bash
# Cursor beforeMCPExecution hook (beta). Reads JSON from stdin; prints allow/deny JSON.
# YAITracker workflow rules are enforced in the MCP server (internal/mcp); this hook is a
# pass-through so Cursor can be extended later without duplicating logic.
set -euo pipefail
cat >/dev/null || true
printf '%s\n' '{"permission":"allow"}'
