package logevent

// Runs the shared cross-SDK conformance fixtures against the Go SDK.
//
// The fixtures in spec/conformance/ are language-neutral: every SDK runs
// the same cases so a behavior that drifts in one lane fails there. See
// spec/conformance/README.md for the fixture schema and for the behaviors
// that are deliberately language-specific (and therefore not pinned).

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type conformanceFixture struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Configure   *struct {
		Service      string `json:"service"`
		DefaultLevel string `json:"defaultLevel"`
	} `json:"configure"`
	Call *struct {
		Fn           string         `json:"fn"`
		Event        string         `json:"event"`
		Level        string         `json:"level"`
		ErrorMessage string         `json:"error_message"`
		Fields       map[string]any `json:"fields"`
	} `json:"call"`
	Expect struct {
		Emitted        *bool          `json:"emitted"`
		ConfigureError bool           `json:"configure_error"`
		Fields         map[string]any `json:"fields"`
		Present        []string       `json:"present"`
		NotFields      map[string]any `json:"not_fields"`
	} `json:"expect"`
}

// resetState returns the package to its pristine post-init state and points
// the sink at buf, so no fixture can leak configuration into the next.
func resetState(buf *bytes.Buffer) {
	mu.Lock()
	defer mu.Unlock()
	service = "unknown"
	defaultLevel = Info
	out = buf
}

func TestConformance(t *testing.T) {
	dir := filepath.Join("..", "spec", "conformance")
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	// A glob matching nothing would make every subtest vacuously pass.
	if len(paths) == 0 {
		t.Fatalf("no conformance fixtures found under %s", dir)
	}

	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var fx conformanceFixture
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		t.Run(fx.Name, func(t *testing.T) {
			var buf bytes.Buffer
			resetState(&buf)

			if fx.Configure != nil {
				opts := ConfigureOpts{Service: fx.Configure.Service, Out: &buf}
				if fx.Configure.DefaultLevel != "" {
					opts.DefaultLevel = LogLevel(fx.Configure.DefaultLevel)
				}
				err := Configure(opts)
				if fx.Expect.ConfigureError {
					if err == nil {
						t.Fatal("expected Configure to return an error, got nil")
					}
					return
				}
				if err != nil {
					t.Fatalf("Configure: %v", err)
				}
			} else if fx.Expect.ConfigureError {
				t.Fatal("a case with configure: null cannot expect a configure error")
			}

			if fx.Call == nil {
				return
			}

			fields := make([]Field, 0, len(fx.Call.Fields))
			for k, v := range fx.Call.Fields {
				fields = append(fields, F(k, v))
			}

			switch fx.Call.Fn {
			case "logEvent":
				LogEvent(fx.Call.Event, LogLevel(fx.Call.Level), fields...)
			case "logError":
				LogError(fx.Call.Event, errors.New(fx.Call.ErrorMessage), fields...)
			default:
				t.Fatalf("unknown fixture fn: %s", fx.Call.Fn)
			}

			if fx.Expect.Emitted != nil && !*fx.Expect.Emitted {
				if buf.Len() != 0 {
					t.Fatalf("expected no output, got %q", buf.String())
				}
				return
			}

			var got map[string]any
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("output is not valid JSON (%v): %q", err, buf.String())
			}

			for k, want := range fx.Expect.Fields {
				if !reflect.DeepEqual(got[k], want) {
					t.Errorf("field %s: got %#v, want %#v", k, got[k], want)
				}
			}
			for _, k := range fx.Expect.Present {
				if _, ok := got[k]; !ok {
					t.Errorf("expected field %s to be present", k)
				}
			}
			for k, unwanted := range fx.Expect.NotFields {
				if reflect.DeepEqual(got[k], unwanted) {
					t.Errorf("field %s must not be %#v", k, unwanted)
				}
			}
		})
	}
}
