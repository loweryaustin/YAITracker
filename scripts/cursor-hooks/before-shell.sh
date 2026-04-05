#!/usr/bin/env bash
# Cursor beforeShellExecution: require begin_work lock (see yait_before_shell.py).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export YAITRACKER_REPO_ROOT="$REPO_ROOT"
exec python3 "$REPO_ROOT/scripts/cursor-hooks/yait_before_shell.py"
