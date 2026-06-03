package logevent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
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
	service      = "unknown"
	defaultLevel = Info
	out          io.Writer = os.Stdout
	hostname     string
	pid          int
)

func init() {
	hostname, _ = os.Hostname()
	pid = os.Getpid()
}

func Configure(opts ConfigureOpts) {
	if opts.Service == "" {
		panic("logevent.Configure: Service must be non-empty")
	}
	mu.Lock()
	defer mu.Unlock()
	service = opts.Service
	if opts.DefaultLevel != "" {
		defaultLevel = opts.DefaultLevel
	}
	if opts.Out != nil {
		out = opts.Out
	}
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
		defer func() { recover() }()

		mu.RLock()
		svc := service
		lvl := level
		if lvl == "" {
			lvl = defaultLevel
		}
		w := out
		mu.RUnlock()

		payload := make(map[string]any, 8+len(fields))
		payload["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
		payload["level"] = string(lvl)
		payload["level_code"] = levelCodes[lvl]
		payload["service"] = svc
		payload["hostname"] = hostname
		payload["pid"] = pid
		payload["event"] = event

		for _, f := range fields {
			payload[f.Key] = f.Value
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "%s\n", data)
	}()
}

func LogError(event string, err error, fields ...Field) {
	extra := make([]Field, 0, len(fields)+3)
	if err != nil {
		extra = append(extra, F("error_type", errorType(err)))
		extra = append(extra, F("error_message", err.Error()))
		extra = append(extra, F("stack", captureStack()))
	}
	extra = append(extra, fields...)
	LogEvent(event, Error, extra...)
}

func errorType(err error) string {
	return fmt.Sprintf("%T", err)
}

func captureStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func GetService() string {
	mu.RLock()
	defer mu.RUnlock()
	return service
}
