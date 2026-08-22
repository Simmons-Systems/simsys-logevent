# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- **BREAKING — Cross-SDK: `configure()` now validates the configured
  default level.** Previously only the per-event level was checked, and
  the "unknown level falls back to the configured default" normalization
  is a no-op when the default is *itself* invalid — so an invalid
  configured default reached the emitter and produced three different
  broken outputs: Node emitted `level:"bogus"` with the `level_code` key
  dropped entirely (`JSON.stringify` omits `undefined`), Python raised
  `KeyError` inside the never-throw guard so the **entire event silently
  vanished**, and Go emitted `level:"bogus"` with `level_code: 0`.
  `configure()` now rejects an invalid default at startup using each
  SDK's existing convention (Node throws, Python raises `ValueError`,
  Go returns an error), reusing the existing level tables as the oracle
  so there is no second list to drift. No state is mutated when
  configuration is rejected.
- **BREAKING — Cross-SDK: whitespace-only service names are rejected,
  and the service name is stored trimmed.** `"   "` previously passed
  every SDK's emptiness check and became a legitimate service identity;
  untrimmed names also split one service across two values in Loki.

### Changed

- **Cross-SDK: unknown level strings normalize to the configured
  default level** in Node, Python, and Go, eliminating the
  level/level_code mismatch (FR-071/FR-075).
- **Cross-SDK: `logError`/`log_error`/`LogError` caller-supplied
  fields now take precedence over extracted error fields** on key
  collision — previously Node and Python silently overwrote them
  (FR-070/FR-085). Go already behaved this way; now documented.
- **Node: version 0.3.0** (behavior changes above).
- **Go: `Configure` returns an error** instead of panicking on empty
  service, matching Node/Python (FR-076/FR-086). `LogEvent`'s panic
  recovery now writes a best-effort diagnostic to stderr instead of
  swallowing silently (FR-072). `captureStack` grows its buffer
  (4 KiB → up to 256 KiB) instead of silently truncating deep stacks
  (FR-087). `errorType` helper inlined (FR-089).
- **Python: `configure()`/`log_event()` state is lock-guarded**
  (FR-073); `log_error`'s `error` parameter annotation widened to
  `Any` to match its documented accept-anything behavior (FR-088);
  packaging metadata added (`python/pyproject.toml`, v0.1.0).
- **CI: Python test workflow added** (pytest, 3.10 floor + 3.13
  lanes) — completes SDK CI coverage alongside the Go workflow
  (FR-090).
- Docs: README event-naming row now documents the dot-separated
  convention as convention-not-enforced (FR-091); CONTRIBUTING
  pre-commit install instructions fixed — Python tooling, not npx
  (FR-092).

## [0.2.0] - 2026-07-05 (`node-v0.2.0`)

### Added

- **Node: `level_code`, `hostname`, and `pid` auto-populated on every
  event** (#15) — `level_code` is an integer level mapping (debug=1,
  info=2, warn=3, error=4) for faster numeric filtering in LogQL;
  `hostname` and `pid` identify the emitting host and process in
  multi-host/multi-process deployments.
- **Node: `logError(event, error, extra?)` convenience helper** (#15) —
  extracts `error_type`, `error_message`, and `stack` from `Error`
  objects; those enrichment fields are documented as suggested optional
  fields on `LogEventPayload`.
- **Python SDK** (`python/simsys_logevent/`): `configure()`,
  `log_event()`, `log_error()`, `get_service()` (#16). Same minimal
  philosophy as Node — structured JSON to stdout, no transports,
  batching, or sampling — with the same auto-populated fields and a
  never-throws contract.
- **Go SDK** (`go/`): `Configure()`, `LogEvent()`, `LogError()`,
  `GetService()`, and the `F()` field builder (#16). Same auto-populated
  fields and a never-panics contract.
- CI/security: CodeQL workflow added; Go CI workflow (go vet + race
  tests + fuzz smoke) with a native `FuzzLogEvent` fuzz target;
  Scorecard workflow findings resolved (#34).

### Changed

- **Node: engines floor raised `node>=18` → `node>=20`; package version
  `0.1.1` → `0.2.0`.** Node 18 has been EOL since April 2025, and the dev
  toolchain (vitest@4) already required Node 20+. CI's Node 20 matrix lane
  now exercises the floor directly, so the build-only `build-node18` job
  was removed. Consumers still on Node 18 should stay on `0.1.x`. No API
  or runtime behavior changes.
- Docs: README and CONTRIBUTING updated for the Python and Go SDKs and
  the auto-populated `level_code`/`hostname`/`pid` fields; stale
  pre-transfer links fixed.

### Security

- **Log-spoofing fix across all three SDKs** (a0d4b87, #30): the
  reserved envelope fields (`ts`, `level`, `level_code`, `service`,
  `hostname`, `pid`, `event`) are now stamped *after* caller-supplied
  extras are merged, so caller fields can no longer override system
  metadata and forge log lines. Consumers pick up this hardening by
  upgrading to this release's tarball.

## [0.1.1] - 2026-04-29 (`node-v0.1.1`)

### Changed

- Repository transferred from `Avicennasis/simsys-logevent` to
  `Simmons-Systems/simsys-logevent`. GitHub redirects keep old URLs
  working, but the new owner is the canonical source. No functional
  changes; consumers should update tarball URLs to the new owner +
  `node-v0.1.1` at their convenience.

## [0.1.0] - 2026-04-27 (`node-v0.1.0`)

Initial release. Node-only.

### Added

- `configure({ service, defaultLevel?, out? })` — set the service name
  once at startup; required before `logEvent()` can run.
- `logEvent({ event, level?, user?, route?, outcome?, ...extras })` —
  emit one JSON line per call. Library auto-stamps `ts` (ISO 8601 UTC)
  and `service` (from `configure()`).
- `getService()` — returns the current service name.
- Schema: `ts`, `level`, `service`, `event` (required) plus suggested
  `user`, `route`, `outcome` and any caller-supplied extras.
- Never-throw guarantees: missing/empty `event` drops silently;
  null/undefined payloads drop; circular references drop (library
  catches `JSON.stringify` failures).
- 11 vitest tests covering the schema, never-throw paths, and
  `out` override behaviour.
- GitHub release tarball distribution (no npm registry).

[Unreleased]: https://github.com/Simmons-Systems/simsys-logevent/compare/node-v0.2.0...HEAD
[0.2.0]: https://github.com/Simmons-Systems/simsys-logevent/releases/tag/node-v0.2.0
[0.1.1]: https://github.com/Simmons-Systems/simsys-logevent/releases/tag/node-v0.1.1
[0.1.0]: https://github.com/Simmons-Systems/simsys-logevent/releases/tag/node-v0.1.0
