#!/usr/bin/env python3
"""Cursor beforeMCPExecution: record active YAIT issue for shell gate; always allow MCP."""
from __future__ import annotations

import json
import os
import sys


def repo_root() -> str:
    return os.environ.get("YAITRACKER_REPO_ROOT") or os.path.dirname(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    )


def lock_path() -> str:
    return os.path.join(repo_root(), ".cursor", "yait-work-lock")


def extract_tool(data: dict) -> str:
    if not isinstance(data, dict):
        return ""
    for key in ("toolName", "tool_name", "mcpToolName", "name"):
        v = data.get(key)
        if isinstance(v, str) and v:
            return v
    req = data.get("request")
    if isinstance(req, dict):
        for key in ("toolName", "name"):
            v = req.get(key)
            if isinstance(v, str) and v:
                return v
    return ""


def extract_args(data: dict) -> dict:
    if not isinstance(data, dict):
        return {}
    for key in ("arguments", "toolArguments", "toolInput", "params", "input"):
        v = data.get(key)
        if isinstance(v, dict):
            return v
    req = data.get("request")
    if isinstance(req, dict):
        return extract_args(req)
    return {}


def allow() -> None:
    print('{"permission":"allow"}', flush=True)


def main() -> None:
    raw = sys.stdin.read()
    try:
        data = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        allow()
        return

    tool = extract_tool(data)
    args = extract_args(data)
    pk = str(args.get("project_key", "")).strip().upper()
    num = args.get("number")

    root = repo_root()
    lock = lock_path()
    os.makedirs(os.path.dirname(lock), exist_ok=True)

    try:
        if tool == "begin_work" and pk == "YAIT" and num is not None:
            n = int(num)
            with open(lock, "w", encoding="utf-8") as f:
                f.write(f"{pk} {n}\n")
        elif tool == "complete_work" and pk == "YAIT" and num is not None:
            want = int(num)
            if os.path.isfile(lock):
                with open(lock, encoding="utf-8") as f:
                    parts = f.read().strip().split()
                if len(parts) >= 2 and parts[0].upper() == pk and int(parts[1]) == want:
                    try:
                        os.remove(lock)
                    except OSError:
                        pass
    except (TypeError, ValueError, OSError):
        pass

    allow()


if __name__ == "__main__":
    main()
