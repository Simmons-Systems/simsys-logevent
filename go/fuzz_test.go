package logevent

// Native Go fuzz target for the JSON log emitter. LogEvent takes
// caller-controlled event names, levels, and field key/values, and its
// contract is: never panic, emit exactly one line of valid JSON per
// non-empty event, and never let caller fields clobber the built-in
// envelope keys.
//
// CI runs the seed corpus on every `go test` plus a short -fuzz smoke;
// run `go test -fuzz=FuzzLogEvent` locally for longer exploration.

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

// fuzzMu serializes fuzz executions: LogEvent writes through shared
// package-level state (Configure's out writer), and parallel fuzz
// workers would otherwise race on the capture buffer.
var fuzzMu sync.Mutex

func FuzzLogEvent(f *testing.F) {
	f.Add("deploy_started", "info", "region", "us-east-1")
	f.Add("", "debug", "k", "v")
	f.Add("weird\x00event", "not-a-level", "ts", "spoofed")
	f.Add("unicode-événement", "error", "event", "clobber-attempt")
	f.Add(strings.Repeat("x", 10000), "warn", "", "")
	f.Fuzz(func(t *testing.T, event, level, fieldKey, fieldValue string) {
		fuzzMu.Lock()
		defer fuzzMu.Unlock()

		var buf bytes.Buffer
		Configure(ConfigureOpts{Service: "fuzz-svc", Out: &buf})

		LogEvent(event, LogLevel(level), F(fieldKey, fieldValue))

		if event == "" {
			if buf.Len() != 0 {
				t.Fatalf("LogEvent with empty event wrote output: %q", buf.String())
			}
			return
		}
		line := buf.String()
		if !strings.HasSuffix(line, "\n") || strings.Count(line, "\n") != 1 {
			t.Fatalf("expected exactly one newline-terminated line, got %q", line)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSuffix(line, "\n")), &payload); err != nil {
			t.Fatalf("output is not valid JSON: %v (line=%q)", err, line)
		}
		// Built-in envelope keys must never be clobbered by caller fields.
		if payload["service"] != "fuzz-svc" {
			t.Fatalf("service clobbered: %v", payload["service"])
		}
		if utf8.ValidString(event) && payload["event"] != event {
			t.Fatalf("event mismatch: got %v, want %q", payload["event"], event)
		}
		// level_code is always one of the known codes (0 for unknown levels
		// because the map lookup zero-values).
		if code, ok := payload["level_code"].(float64); !ok || code < 0 || code > 4 {
			t.Fatalf("level_code out of range: %v", payload["level_code"])
		}
	})
}
