# Cross-SDK conformance fixtures

Language-neutral test vectors. Every SDK runs the same cases, so a
behavior that drifts in one lane fails in that lane instead of being
discovered later in Loki.

The real product of this repo is the event contract, not any one
library. These fixtures are that contract in executable form.

| Lane | Runner |
|---|---|
| Node | `node/tests/conformance.test.ts` |
| Python | `python/tests/test_conformance.py` |
| Go | `go/conformance_test.go` |

## Fixture schema

One JSON file per case.

```jsonc
{
  "name": "basic-event",
  "description": "Why this case exists and what regressing it would mean.",

  // Config to apply. `null` means "do not call configure()" — the case
  // observes the SDK in its default state.
  "configure": { "service": "test-svc", "defaultLevel": "warn" },

  // Optional. Omit for configure-only cases.
  "call": {
    "fn": "logEvent",            // or "logError"
    "event": "demo.event",
    "level": "bogus",            // optional per-event level
    "error_message": "boom",     // logError only
    "fields": { "outcome": "success" }
  },

  "expect": {
    "configure_error": true,     // configure() must reject; nothing else runs
    "emitted": true,             // false = must emit nothing, without raising
    "fields":     { "level": "warn", "level_code": 3 },  // exact match
    "present":    ["ts", "hostname", "pid"],             // key exists, value unchecked
    "not_fields": { "service": "fake-svc" }              // must NOT equal
  }
}
```

`present` exists for values that are real but not fixable across lanes
(timestamps, hostname, pid, and `error_type`, which is `Error` in Node,
`ValueError` in Python and `*errors.errorString` in Go).

## Adding a case

1. Write the fixture.
2. **Prove it discriminates.** Run it against code that lacks the
   behavior and watch it fail in all three lanes. A case that passes on
   the broken build measures nothing — and is worse than no case,
   because it reads as coverage.
3. Confirm it passes on the fixed code in all three lanes.

Every case currently here was validated that way: reverting the three
SDK sources to their pre-fix state fails exactly
`invalid-configured-default`, `whitespace-service`, `service-is-trimmed`
and `service-inner-whitespace-preserved` — the same four, in all three
lanes.

## Deliberately NOT pinned

These genuinely differ per language. Pinning them would either force an
unnatural implementation or produce a fixture that cannot pass
everywhere. They are recorded here so the divergence is *documented*
rather than merely undiscovered.

**Unserializable field values** — measured, same input, three answers:

| Field value | Node | Python | Go |
|---|---|---|---|
| function | key dropped, event still emitted | stringified via `default=str` | **whole event dropped** |
| big integer (10³⁰) | **whole event dropped** (BigInt throws) | emitted as an oversized JSON number | n/a (int64-bounded) |
| circular reference | whole event dropped | whole event dropped | n/a |

**`ts` precision** — Node emits milliseconds, Python microseconds, Go
nanoseconds (`time.RFC3339Nano`). All are ISO-8601 UTC ending in `Z`,
and LogQL parses all three, so only the digit count differs.

**`level` is a caller input, not a library-owned field.** It is set by
the documented `level` option, so supplying it is legitimate rather than
spoofing. `level_code` *is* library-owned: always derived from the
resolved level, never settable. See `reserved-field-spoof.json`.

## Related

The sibling repo `simsys-metrics` uses `spec/` and JSON too, but its
artifact is a static catalogue (metric names, types, labels, bucket
schedules) rather than input/expected-output vectors — a catalogue
cannot express "the never-throw guard swallowed this", and these vectors
cannot express a bucket schedule. The two agree on the `service`
dimension that joins them, and deliberately stop there.

Note that `service` semantics differ between the repos under
misconfiguration: logevent falls back to `service: "unknown"` so logging
survives a broken bootstrap, whereas metrics treats service identity as
mandatory. Both trim, so the values agree on every non-pathological
path.
