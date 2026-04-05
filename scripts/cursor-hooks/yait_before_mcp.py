#!/usr/bin/env python3
"""Cursor beforeMCPExecution: relay conversation_id to Sidecar and manage work lock."""
from __future__ import annotations

import json
import os
import sys
import tempfile


def repo_root() -> str:
    return os.environ.get("YAITRACKER_REPO_ROOT") or os.path.dirname(
        os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    )


def relay_conversation_id(data: dict) -> None:
    """Write conversation_id to a file the Sidecar reads before each tool call."""
    conv_id = data.get("conversation_id", "")
    if not conv_id:
        return
    target = os.path.join(repo_root(), ".cursor", "yait-conversation-id")
    try:
        fd, tmp = tempfile.mkstemp(dir=os.path.dirname(target))
        try:
            os.write(fd, conv_id.encode("utf-8"))
        finally:
            os.close(fd)
        os.replace(tmp, target)
    except OSError:
        pass


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
    for key in ("arguments", "toolArguments", "tool_input", "toolInput", "params", "input"):
        v = data.get(key)
        if isinstance(v, dict):
            return v
        if isinstance(v, str) and v.strip().startswith("{"):
            try:
                parsed = json.loads(v)
                if isinstance(parsed, dict):
                    return parsed
            except json.JSONDecodeError:
                pass
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

    relay_conversation_id(data)

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
