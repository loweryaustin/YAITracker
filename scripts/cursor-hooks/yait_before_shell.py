#!/usr/bin/env python3
"""Cursor beforeShellExecution: require YAIT begin_work (lock file) unless skipped or allowlisted."""
from __future__ import annotations

import json
import os
import re
import sys


def repo_root() -> str:
    return os.environ.get("YAITRACKER_REPO_ROOT") or os.path.dirname(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    )


def lock_path() -> str:
    return os.path.join(repo_root(), ".cursor", "yait-work-lock")


def extract_command(data: dict) -> str:
    if not isinstance(data, dict):
        return ""
    for key in ("command", "commandLine", "cmd", "shell_command"):
        v = data.get(key)
        if isinstance(v, str):
            return v
    return ""


def deny(msg: str) -> None:
    print(
        json.dumps(
            {
                "permission": "deny",
                "user_message": msg,
                "agent_message": msg,
            }
        ),
        flush=True,
    )


def allow() -> None:
    print('{"permission":"allow"}', flush=True)


# Read-only / meta commands that do not require an active begin_work lock.
_READ_ONLY_RE = re.compile(
    r"^\s*(git\s+(status|diff|log|show|branch|rev-parse|describe)(\s|$)|pwd\s*$|whoami\s*$|"
    r"go\s+version\s*$|which\s+|env\s*$|printenv\s*$|echo\s+|true\s*$|false\s*$|:\s*$)",
    re.IGNORECASE,
)


def main() -> None:
    if os.environ.get("YAIT_HOOK_SKIP") == "1":
        allow()
        return

    root = repo_root()
    disable = os.path.join(root, ".cursor", "yait-hook-disable")
    if os.path.isfile(disable):
        allow()
        return

    raw = sys.stdin.read()
    try:
        data = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        allow()
        return

    cmd = extract_command(data)
    if cmd and _READ_ONLY_RE.match(cmd):
        allow()
        return

    if os.path.isfile(lock_path()):
        allow()
        return

    deny(
        "Shell blocked: call YAITracker begin_work(project_key: YAIT, number: N) first, "
        "or set YAIT_HOOK_SKIP=1, or add file .cursor/yait-hook-disable (emergency only). "
        "See docs/mcp-agent-workflow.md."
    )


if __name__ == "__main__":
    main()
