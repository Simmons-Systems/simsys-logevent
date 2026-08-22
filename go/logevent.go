package logevent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

var levelCodes = map[LogLevel]int{
	Debug: 1,
	Info:  2,
	Warn:  3,
	Error: 4,
}

type ConfigureOpts struct {
	Service      string
	DefaultLevel LogLevel
	Out          io.Writer
}

var (
	mu           sync.RWMutex
	service                = "unknown"
	defaultLevel           = Info
	out          io.Writer = os.Stdout
	hostname     string
	pid          int
)

func init() {
	hostname, _ = os.Hostname()
	pid = os.Getpid()
}

// Configure sets the module-level state. Typically called once at startup.
//
// Returns an error when Service is empty or whitespace-only, or when
// DefaultLevel is set to something outside debug/info/warn/error — matching
// the Node and Python SDKs, which reject both without crashing the process.
// Service is stored trimmed. No state is mutated when an error is returned.
// (Changed from panicking; see ticket FR-076/FR-086.)
func Configure(opts ConfigureOpts) error {
	svc := strings.TrimSpace(opts.Service)
	if svc == "" {
		return fmt.Errorf("logevent.Configure: Service must be non-empty")
	}
	// LogEvent normalizes an unknown level *to the default*, which is a no-op
	// when the default is itself invalid — the event then carries a bogus
	// level with level_code 0 (the map zero value). Reject it at startup,
	// before any state is mutated, where it is diagnosable.
	if opts.DefaultLevel != "" {
		if _, ok := levelCodes[opts.DefaultLevel]; !ok {
			return fmt.Errorf("logevent.Configure: DefaultLevel %q is not a valid level", opts.DefaultLevel)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	service = svc
	if opts.DefaultLevel != "" {
		defaultLevel = opts.DefaultLevel
	}
	if opts.Out != nil {
		out = opts.Out
	}
	return nil
}

type Field struct {
	Key   string
	Value any
}

func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func LogEvent(event string, level LogLevel, fields ...Field) {
	if event == "" {
		return
	}
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Logging must never crash the caller, but swallowing
				// panics with zero output made failures undiagnosable
				// (FR-072). Best-effort breadcrumb to stderr.
				fmt.Fprintf(os.Stderr, "simsys-logevent: recovered panic in LogEvent(%q): %v\n", event, r)
			}
		}()

		mu.RLock()
		svc := service
		lvl := level
		if lvl == "" {
			lvl = defaultLevel
		}
		if _, ok := levelCodes[lvl]; !ok {
			// Unknown level strings previously produced a level/level_code
			// mismatch (level kept the bogus string, level_code was 0).
			// Normalize to the configured default so the pair always
			// agrees (FR-071).
			lvl = defaultLevel
		}
		w := out
		mu.RUnlock()

		payload := make(map[string]any, 8+len(fields))

		for _, f := range fields {
			payload[f.Key] = f.Value
		}

		payload["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
		payload["level"] = string(lvl)
		payload["level_code"] = levelCodes[lvl]
		payload["service"] = svc
		payload["hostname"] = hostname
		payload["pid"] = pid
		payload["event"] = event

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "%s\n", data)
	}()
}

// LogError emits an error-level event with error_type, error_message, and
// stack extracted from err. Caller-supplied fields take precedence over the
// extracted ones on key collision.
func LogError(event string, err error, fields ...Field) {
	extra := make([]Field, 0, len(fields)+3)
	if err != nil {
		extra = append(extra, F("error_type", fmt.Sprintf("%T", err)))
		extra = append(extra, F("error_message", err.Error()))
		extra = append(extra, F("stack", captureStack()))
	}
	// User fields append last: in LogEvent's payload map, later duplicate
	// keys win, so explicit caller fields override extracted error fields.
	extra = append(extra, fields...)
	LogEvent(event, Error, extra...)
}

// captureStack returns the current goroutine's stack, growing the buffer
// until the trace fits (previously a fixed 4096-byte buffer silently
// truncated deep stacks — FR-087). Capped at 256 KiB.
func captureStack() string {
	size := 4096
	for {
		buf := make([]byte, size)
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		size *= 2
		if size > 256*1024 {
			return string(buf[:n]) + "\n... (stack truncated at 256KiB)"
		}
	}
}

func GetService() string {
	mu.RLock()
	defer mu.RUnlock()
	return service
}
