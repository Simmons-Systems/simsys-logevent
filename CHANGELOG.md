# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Simmons-Systems/simsys-logevent/compare/node-v0.1.1...HEAD
[0.1.1]: https://github.com/Simmons-Systems/simsys-logevent/releases/tag/node-v0.1.1
[0.1.0]: https://github.com/Simmons-Systems/simsys-logevent/releases/tag/node-v0.1.0
