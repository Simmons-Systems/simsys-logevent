package logevent

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func setup(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	if err := Configure(ConfigureOpts{Service: "test-svc", Out: &buf}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	return &buf
}

func TestConfigureRejectsEmptyService(t *testing.T) {
	if err := Configure(ConfigureOpts{Service: ""}); err == nil {
		t.Fatal("Configure with empty Service: expected error, got nil")
	}
}

func TestConfigureRejectsWhitespaceService(t *testing.T) {
	if err := Configure(ConfigureOpts{Service: "   "}); err == nil {
		t.Fatal("Configure with whitespace-only Service: expected error, got nil")
	}
}

func TestConfigureTrimsService(t *testing.T) {
	var buf bytes.Buffer
	if err := Configure(ConfigureOpts{Service: "  spaced  ", Out: &buf}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if got := GetService(); got != "spaced" {
		t.Fatalf("GetService() = %q, want %q", got, "spaced")
	}
}

func TestConfigureRejectsInvalidDefaultLevel(t *testing.T) {
	err := Configure(ConfigureOpts{Service: "test-svc", DefaultLevel: LogLevel("bogus")})
	if err == nil {
		t.Fatal("Configure with invalid DefaultLevel: expected error, got nil")
	}
}

func TestUnknownLevelNormalizesToDefault(t *testing.T) {
	buf := setup(t)
	LogEvent("demo.event", LogLevel("bogus"))
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed["level"] != "info" {
		t.Errorf("level = %v, want info (normalized default)", parsed["level"])
	}
	if parsed["level_code"] != float64(2) {
		t.Errorf("level_code = %v, want 2", parsed["level_code"])
	}
}

func TestLogErrorCallerFieldsWin(t *testing.T) {
	buf := setup(t)
	LogError("db.fail", errors.New("boom"), F("error_message", "caller override"))
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed["error_message"] != "caller override" {
		t.Errorf("error_message = %v, want caller override", parsed["error_message"])
	}
	if parsed["error_type"] != "*errors.errorString" {
		t.Errorf("error_type = %v, want *errors.errorString", parsed["error_type"])
	}
}

func TestEmitsOneJSONLine(t *testing.T) {
	buf := setup(t)
	LogEvent("demo.event", Info)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if m["event"] != "demo.event" {
		t.Errorf("event = %v, want demo.event", m["event"])
	}
}

func TestIncludesCoreFields(t *testing.T) {
	buf := setup(t)
	LogEvent("demo", Info)
	var m map[string]any
	json.Unmarshal(buf.Bytes(), &m)
	for _, key := range []string{"ts", "level", "level_code", "service", "hostname", "pid"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing field %q", key)
		}
	}
	if m["level"] != "info" {
		t.Errorf("level = %v, want info", m["level"])
	}
	if m["service"] != "test-svc" {
		t.Errorf("service = %v, want test-svc", m["service"])
	}
}

func TestLevelCodes(t *testing.T) {
	buf := setup(t)
	for _, tc := range []struct {
		lvl  LogLevel
		code float64
	}{
		{Debug, 1}, {Info, 2}, {Warn, 3}, {Error, 4},
	} {
		buf.Reset()
		LogEvent("demo", tc.lvl)
		var m map[string]any
		json.Unmarshal(buf.Bytes(), &m)
		if m["level_code"] != tc.code {
			t.Errorf("level_code for %s = %v, want %v", tc.lvl, m["level_code"], tc.code)
		}
	}
}

func TestExtraFields(t *testing.T) {
	buf := setup(t)
	LogEvent("shift", Info, F("user", "alice"), F("outcome", "success"))
	var m map[string]any
	json.Unmarshal(buf.Bytes(), &m)
	if m["user"] != "alice" {
		t.Errorf("user = %v, want alice", m["user"])
	}
	if m["outcome"] != "success" {
		t.Errorf("outcome = %v, want success", m["outcome"])
	}
}

func TestEmptyEventSkipped(t *testing.T) {
	buf := setup(t)
	LogEvent("", Info)
	if buf.Len() > 0 {
		t.Error("expected no output for empty event")
	}
}

func TestNeverPanics(t *testing.T) {
	setup(t)
	LogEvent("demo", Info, F("fn", func() {}))
}

func TestLogError(t *testing.T) {
	buf := setup(t)
	LogError("db.failed", errors.New("connection reset"))
	var m map[string]any
	json.Unmarshal(buf.Bytes(), &m)
	if m["level"] != "error" {
		t.Errorf("level = %v, want error", m["level"])
	}
	if m["error_type"] != "*errors.errorString" {
		t.Errorf("error_type = %v", m["error_type"])
	}
	if m["error_message"] != "connection reset" {
		t.Errorf("error_message = %v", m["error_message"])
	}
	if _, ok := m["stack"]; !ok {
		t.Error("missing stack field")
	}
}

func TestLogErrorWithExtraFields(t *testing.T) {
	buf := setup(t)
	LogError("api.failed", errors.New("timeout"), F("route", "/api/data"))
	var m map[string]any
	json.Unmarshal(buf.Bytes(), &m)
	if m["route"] != "/api/data" {
		t.Errorf("route = %v, want /api/data", m["route"])
	}
}

func TestGetService(t *testing.T) {
	setup(t)
	if s := GetService(); s != "test-svc" {
		t.Errorf("GetService() = %q, want test-svc", s)
	}
}

func TestLogEvent_NoSpoofing(t *testing.T) {
	var buf bytes.Buffer
	if err := Configure(ConfigureOpts{
		Service: "test-service",
		Out:     &buf,
	}); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	LogEvent("auth.signin", Info, F("service", "hacker"), F("pid", 999), F("event", "spoofed"))

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	if parsed["service"] != "test-service" {
		t.Errorf("expected service 'test-service', got '%v'", parsed["service"])
	}
	if parsed["event"] != "auth.signin" {
		t.Errorf("expected event 'auth.signin', got '%v'", parsed["event"])
	}
}
