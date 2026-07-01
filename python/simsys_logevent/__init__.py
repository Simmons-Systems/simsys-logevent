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
import traceback
import time
from datetime import datetime, timezone
from typing import Any, Callable, Literal, Optional

LogLevel = Literal["debug", "info", "warn", "error"]

_LEVEL_CODES: dict[str, int] = {
    "debug": 1,
    "info": 2,
    "warn": 3,
    "error": 4,
}

_service: str = "unknown"
_default_level: LogLevel = "info"
_out: Callable[[str], Any] = lambda line: sys.stdout.write(line + "\n")
_hostname: str = socket.gethostname()
_pid: int = os.getpid()

# Cache the current ISO string to avoid expensive datetime formatting
# on every log event. Valid since time.time() has high resolution but
# log timestamps only need millisecond precision here.
_last_time: int = 0
_last_iso: str = ""

def _get_iso_timestamp() -> str:
    global _last_time, _last_iso
    now_ms = int(time.time() * 1000)
    if now_ms != _last_time:
        # Generate datetime from the exact same timestamp to prevent drift
        dt = datetime.fromtimestamp(now_ms / 1000.0, timezone.utc)
        new_iso = dt.isoformat(timespec="milliseconds").replace("+00:00", "Z")
        _last_iso = new_iso
        _last_time = now_ms
    return _last_iso


def configure(
    service: str,
    default_level: LogLevel = "info",
    out: Optional[Callable[[str], Any]] = None,
) -> None:
    global _service, _default_level, _out
    if not service or not isinstance(service, str):
        raise ValueError("configure() requires a non-empty service string.")
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
    lvl = level or _default_level
    try:
        payload = {
            "ts": _get_iso_timestamp(),
            "level": lvl,
            "level_code": _LEVEL_CODES.get(lvl, _LEVEL_CODES[_default_level]),
            "service": _service,
            "hostname": _hostname,
            "pid": _pid,
            "event": event,
            **kwargs,
        }
        _out(json.dumps(payload, default=str))
    except Exception:
        pass


def log_error(
    event: str,
    error: Optional[BaseException] = None,
    **kwargs: Any,
) -> None:
    fields: dict[str, Any] = dict(kwargs)
    if isinstance(error, BaseException):
        fields["error_type"] = type(error).__name__
        fields["error_message"] = str(error)
        fields["stack"] = "".join(traceback.format_exception(error))
    elif error is not None:
        fields["error_message"] = str(error)
    log_event(event, level="error", **fields)


def get_service() -> str:
    return _service
