# simsys-logevent (Python)

Structured JSON log events for Python apps, designed for
systemd-journal → Loki pipelines (Grafana Alloy `loki.source.journal`).
Every call writes one JSON line to stdout.

Intentionally minimal — a stable schema, a configured service name, and
never-throws guarantees. No transports, no batching, no sampling. Loki
and LogQL handle querying.

This is the Python lane of a three-SDK repo (Node, Python, Go) that all
emit the same event envelope. Full documentation, including the shared
schema and cardinality guidance, lives in the
[repository README](https://github.com/Simmons-Systems/simsys-logevent#readme).

## Install

Not yet on PyPI. Install from the repository subdirectory:

```bash
pip install "simsys-logevent @ git+https://github.com/Simmons-Systems/simsys-logevent.git#subdirectory=python"
```

## Usage

```python
from simsys_logevent import configure, log_event, log_error

configure(service="board-portal")

log_event(
    event="auth.signin",
    user="u_8f2c1a94",          # stable internal ID, not an email
    route="/api/auth/callback/google",
    outcome="success",
)

try:
    ...
except Exception as err:
    log_error("db.query.failed", err, route="/api/shifts")
```

Emits one JSON line per call:

```json
{"user": "u_8f2c1a94", "route": "/api/auth/callback/google", "outcome": "success", "ts": "2026-04-27T12:34:56.789Z", "level": "info", "level_code": 2, "service": "board-portal", "hostname": "bfr", "pid": 12345, "event": "auth.signin"}
```

## Schema

Every emitted object includes `ts`, `level`, `level_code`, `service`,
`hostname`, `pid`, and `event`. Library-set fields always win over
caller-supplied keys of the same name, so payloads cannot spoof system
metadata.

Level codes are `debug=1`, `info=2`, `warn=3`, `error=4`.

## Configuration

`configure()` is the only function that raises. It rejects a missing,
non-string, empty, or whitespace-only `service`, and a `default_level`
outside `debug`/`info`/`warn`/`error`. Nothing is mutated when
configuration is rejected, and the service name is stored trimmed.

An invalid **per-event** level is not an error — it normalizes to the
configured default, so a bad level never costs you the log line.

`service` defaults to `"unknown"`, and logging before `configure()`
emits normally rather than failing: a misconfigured bootstrap is exactly
when logs matter most. Treat `{service="unknown"}` in Loki as a bug
signal.

## Sensitive data

This library preserves arbitrary caller fields verbatim and performs no
redaction. Never log passwords, API secrets, private keys, tokens,
session cookies, `Authorization` headers, or full request bodies. Prefer
a stable internal user ID over an email address — `log_error()` also
copies the exception message and full traceback into the event, and
neither is guaranteed free of sensitive data.

## License

MIT. See [LICENSE](https://github.com/Simmons-Systems/simsys-logevent/blob/main/LICENSE).
