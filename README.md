# @simsys/logevent

[![CI](https://github.com/Avicennasis/simsys-logevent/actions/workflows/test-node.yml/badge.svg)](https://github.com/Avicennasis/simsys-logevent/actions/workflows/test-node.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/Avicennasis/simsys-logevent/badge)](https://scorecard.dev/viewer/?uri=github.com/Avicennasis/simsys-logevent)
[![Release](https://img.shields.io/github/v/release/Avicennasis/simsys-logevent?display_name=tag)](https://github.com/Avicennasis/simsys-logevent/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

Structured JSON log events for Node.js, designed for systemd-journal → Loki
pipelines (Grafana Alloy `loki.source.journal`). Every call writes one
JSON line to stdout.

The library is intentionally minimal — `console.log(JSON.stringify(...))`
with a stable schema, a configured service name, and never-throws
guarantees. No transports, no batching, no sampling. Loki and LogQL
handle querying.

## Install

Pin to a release artifact (no npm registry):

```json
{
  "dependencies": {
    "@simsys/logevent": "https://github.com/Avicennasis/simsys-logevent/releases/download/node-v0.1.0/simsys-logevent-0.1.0.tgz"
  }
}
```

## Usage

```ts
import { configure, logEvent } from "@simsys/logevent";

configure({ service: "board-portal" });

logEvent({
  event: "auth.signin",
  user: "alice@example.org",
  route: "/api/auth/callback/google",
  outcome: "success",
});

logEvent({
  event: "shift.assigned",
  user: "admin@bfr.org",
  outcome: "success",
  shift_id: "abc123",
  level: "info",
});
```

Emits one JSON line per call, e.g.:

```json
{"ts":"2026-04-27T12:34:56.789Z","level":"info","service":"board-portal","event":"auth.signin","user":"alice@example.org","route":"/api/auth/callback/google","outcome":"success"}
```

## Schema

Every emitted object includes:

| Field     | Type   | Notes                                                   |
| --------- | ------ | ------------------------------------------------------- |
| `ts`      | string | ISO-8601 UTC. Set by the library.                       |
| `level`   | string | `debug`/`info`/`warn`/`error`. Defaults to `info`.      |
| `service` | string | Set by `configure({ service })`. Required.              |
| `event`   | string | Caller-provided. Dot-separated kebab.                   |

Suggested optional fields:

| Field     | Type   | Notes                                                   |
| --------- | ------ | ------------------------------------------------------- |
| `user`    | string | Email or UID. Free-form; not bounded by the library.    |
| `route`   | string | HTTP route or logical action surface.                   |
| `outcome` | string | `success` / `failure` / `blocked` / domain-specific.    |

Anything else is preserved on the emitted JSON. Use whatever
event-specific fields you need.

## Cardinality reminder

The output is the **log body**, not Loki labels. LogQL `| json` parses
it at query time — so high-cardinality fields (user IDs, free-form
text) stay out of the index. Don't promote any of these to Alloy
relabel rules.

## Configuration

```ts
configure({
  service: "board-portal",      // required
  defaultLevel: "info",         // optional; default "info"
  out: (line) => console.log(line),  // optional; default process.stdout
});
```

The `out` override is for tests. Production code should leave it alone.

## Never-throw guarantees

`logEvent()` swallows:

- missing/empty `event`
- null/undefined payloads
- circular references in the payload

Logging must never crash a request handler — these guards make that
contract explicit.

## License

MIT. See [LICENSE](./LICENSE).
