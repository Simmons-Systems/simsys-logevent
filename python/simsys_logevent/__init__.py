"""simsys-logevent — Structured log events for Python apps.

Writes JSON-per-line to stdout. Designed for systemd-journal -> Loki
pipelines (Grafana Alloy loki.source.journal). Every event becomes a
single LogQL-queryable JSON line.

Public API::

    from simsys_logevent import configure, log_event, log_error

    configure(service="board-portal")
    log_event(event="auth.signin", user="alice@example.org", outcome="success")
    log_error("db.query.failed", error)
"""

from __future__ import annotations

import json
import os
import socket
import sys
import threading
import traceback
from datetime import datetime, timezone
from typing import Any, Callable, Literal, Optional

LogLevel = Literal["debug", "info", "warn", "error"]

_LEVEL_CODES: dict[str, int] = {
    "debug": 1,
    "info": 2,
    "warn": 3,
    "error": 4,
}

# Module-level state guarded by _CONFIG_LOCK (FR-073: the Go SDK protects
# its equivalents with an RWMutex; Python relied on the GIL for atomicity
# of single assignments, which is true but left configure()'s multi-field
# update non-atomic — a concurrent log_event could observe a new service
# with the old sink).
_CONFIG_LOCK = threading.Lock()
_service: str = "unknown"
_default_level: LogLevel = "info"
_out: Callable[[str], Any] = lambda line: sys.stdout.write(line + "\n")
_hostname: str = socket.gethostname()
_pid: int = os.getpid()


def configure(
    service: str,
    default_level: LogLevel = "info",
    out: Optional[Callable[[str], Any]] = None,
) -> None:
    global _service, _default_level, _out
    if not service or not isinstance(service, str):
        raise ValueError("configure() requires a non-empty service string.")
    with _CONFIG_LOCK:
        _service = service
        _default_level = default_level
        if out is not None:
            _out = out


def log_event(
    event: str,
    level: Optional[LogLevel] = None,
    **kwargs: Any,
) -> None:
    if not event or not isinstance(event, str):
        return
    with _CONFIG_LOCK:
        svc = _service
        default_level = _default_level
        out = _out
    lvl = level or default_level
    if lvl not in _LEVEL_CODES:
        # Unknown level strings previously produced a level/level_code
        # mismatch (level kept the bogus string, level_code fell back to
        # the default's code). Normalize both to the configured default
        # so the pair always agrees (FR-071).
        lvl = default_level
    try:
        payload = {
            **kwargs,
            "ts": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            "level": lvl,
            "level_code": _LEVEL_CODES[lvl],
            "service": svc,
            "hostname": _hostname,
            "pid": _pid,
            "event": event,
        }
        out(json.dumps(payload, default=str))
    except Exception:
        pass


def log_error(
    event: str,
    error: Any = None,
    **kwargs: Any,
) -> None:
    """Emit an error-level event with fields extracted from ``error``.

    ``error`` may be an exception (full error_type/error_message/stack
    extraction), any other non-None value (stringified into
    error_message — matching the Node SDK's ``unknown``), or None.

    Caller-supplied ``kwargs`` take precedence over the extracted error
    fields on key collision (previously the extraction silently
    overwrote them — FR-070/FR-085).
    """
    fields: dict[str, Any] = {}
    if isinstance(error, BaseException):
        fields["error_type"] = type(error).__name__
        fields["error_message"] = str(error)
        fields["stack"] = "".join(traceback.format_exception(error))
    elif error is not None:
        fields["error_message"] = str(error)
    fields.update(kwargs)
    log_event(event, level="error", **fields)


def get_service() -> str:
    return _service
