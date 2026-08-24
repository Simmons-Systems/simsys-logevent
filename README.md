# @simsys/logevent

[![CI](https://github.com/Simmons-Systems/simsys-logevent/actions/workflows/test-node.yml/badge.svg)](https://github.com/Simmons-Systems/simsys-logevent/actions/workflows/test-node.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/Simmons-Systems/simsys-logevent/badge)](https://scorecard.dev/viewer/?uri=github.com/Simmons-Systems/simsys-logevent)
[![Release](https://img.shields.io/github/v/release/Simmons-Systems/simsys-logevent?display_name=tag)](https://github.com/Simmons-Systems/simsys-logevent/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

Structured JSON log events for Node.js, Python, and Go, designed for
systemd-journal → Loki pipelines (Grafana Alloy `loki.source.journal`).
Every call writes one JSON line to stdout.

The library is intentionally minimal — `console.log(JSON.stringify(...))`
with a stable schema, a configured service name, and never-throws
guarantees. No transports, no batching, no sampling. Loki and LogQL
handle querying.

## Install

**Node** — pin to a release artifact (not yet on the npm registry):

```json
{
  "dependencies": {
    "@simsys/logevent": "https://github.com/Simmons-Systems/simsys-logevent/releases/download/node-v1.0.0/simsys-logevent-1.0.0.tgz"
  }
}
```

**Go** — consumable directly by module path; the tag is the release:

```bash
go get github.com/Simmons-Systems/simsys-logevent/go@v0.1.0
```

**Python** — packaged with `python/pyproject.toml`, but not yet on PyPI.
Install from the repo subdirectory:

```bash
pip install "simsys-logevent @ git+https://github.com/Simmons-Systems/simsys-logevent.git#subdirectory=python"
```

All three lanes are published from this repo. None are on a public
package index yet — npm and PyPI publication is planned, at which point
the Node and Python snippets above become plain `npm install` /
`pip install`.

## Usage

```ts
import { configure, logEvent, logError } from "@simsys/logevent";

configure({ service: "board-portal" });

logEvent({
  event: "auth.signin",
  user: "u_8f2c1a94",           // stable internal ID, not an email
  route: "/api/auth/callback/google",
  outcome: "success",
});

logEvent({
  event: "shift.assigned",
  user: "u_3d70b2fe",
  outcome: "success",
  shift_id: "abc123",
  level: "info",
});

// In a catch block — extracts error_type/error_message/stack,
// emits at level "error":
logError("db.query.failed", err, { route: "/api/shifts" });
```

Emits one JSON line per call, e.g.:

```json
{"user":"u_8f2c1a94","route":"/api/auth/callback/google","outcome":"success","ts":"2026-04-27T12:34:56.789Z","level":"info","level_code":2,"service":"board-portal","hostname":"bfr","pid":12345,"event":"auth.signin"}
```

## Schema

Every emitted object includes:

| Field        | Type   | Notes                                                |
| ------------ | ------ | ---------------------------------------------------- |
| `ts`         | string | ISO-8601 UTC. Set by the library.                    |
| `level`      | string | `debug`/`info`/`warn`/`error`. Defaults to `info`; unknown strings normalize to the configured default. |
| `level_code` | number | `1`–`4` (debug→error). Set by the library.           |
| `service`    | string | Set by `configure({ service })`. Falls back to `"unknown"` — see below. |
| `hostname`   | string | OS hostname. Set by the library.                     |
| `pid`        | number | Process ID. Set by the library.                      |
| `event`      | string | Caller-provided. Dot-separated kebab by convention (e.g. `auth.signin`) — not enforced by the library. |

Suggested optional fields:

| Field     | Type   | Notes                                                   |
| --------- | ------ | ------------------------------------------------------- |
| `user`    | string | Prefer a stable internal ID over an email. Free-form; not bounded by the library. |
| `route`   | string | HTTP route or logical action surface.                   |
| `outcome` | string | `success` / `failure` / `blocked` / domain-specific.    |

Anything else is preserved on the emitted JSON. Use whatever
event-specific fields you need. Library-set fields (`ts`, `level`,
`level_code`, `service`, `hostname`, `pid`, `event`) always win over
caller-supplied keys of the same name — payloads can't spoof system
metadata.

## Cardinality reminder

The output is the **log body**, not Loki labels. LogQL `| json` parses
it at query time — so high-cardinality fields (user IDs, free-form
text) stay out of the index. Don't promote any of these to Alloy
relabel rules.

## Sensitive data

This library deliberately accepts arbitrary caller fields and preserves
them verbatim. It does **not** redact anything, and it will not grow a
redaction facility — that machinery is exactly the kind of thing this
package exists to avoid. The trust boundary is therefore yours:

**Never log:**

- passwords, API secrets, private keys
- access/refresh tokens, session cookies, `Authorization` headers
- full request bodies or unfiltered exception metadata

**Prefer** a stable internal user ID over an email address. Logs live
longer and travel further than the system that produced them, and an
identifier you control can be re-mapped later; an email cannot.

Error events deserve particular care: `logError()` copies
`error.message` and the full stack into the event, and neither is
guaranteed free of credentials, query fragments, or user data.

## Configuration

```ts
configure({
  service: "board-portal",      // required, non-empty; stored trimmed
  defaultLevel: "info",         // optional; must be debug|info|warn|error
  out: (line) => console.log(line),  // optional; default process.stdout
});
```

`configure()` is the only function that reports errors, and it rejects:

- a missing, non-string, empty, or whitespace-only `service`
- a `defaultLevel` outside `debug`/`info`/`warn`/`error`

Rejection happens before any state is mutated, so a failed `configure()`
leaves the previous configuration intact. The service name is stored
trimmed, so `"  portal  "` and `"portal"` cannot become two different
service identities in Loki.

An invalid **per-event** level is not an error — it normalizes to the
configured default, so a bad level never costs you the log line.

The `out` override is for tests. Production code should leave it alone.

### Logging before `configure()`

`service` defaults to `"unknown"`, and logging before `configure()` runs
emits normally with `service: "unknown"` rather than failing or dropping
the event. This is deliberate: a misconfigured bootstrap is exactly when
logs matter most, so logging never disappears solely because
`configure()` had not been reached yet.

Treat `{service="unknown"}` in Loki as a bug signal — it means some
process is emitting before it configures itself.

## Never-throw guarantees

`logEvent()` swallows:

- missing/empty `event`
- null/undefined payloads
- circular references in the payload

Logging must never crash a request handler — these guards make that
contract explicit.

## License

MIT. See [LICENSE](./LICENSE).
