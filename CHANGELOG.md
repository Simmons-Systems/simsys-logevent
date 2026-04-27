# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Avicennasis/simsys-logevent/compare/node-v0.1.0...HEAD
[0.1.0]: https://github.com/Avicennasis/simsys-logevent/releases/tag/node-v0.1.0
