#!/usr/bin/env bash
# Cursor beforeMCPExecution: record YAIT issue lock on begin_work; clear on complete_work.
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export YAITRACKER_REPO_ROOT="$REPO_ROOT"
exec python3 "$REPO_ROOT/scripts/cursor-hooks/yait_before_mcp.py"
