package config

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type testConfig struct {
	Duration time.Duration `yaml:"duration"`
	Nested   testNested    `yaml:"nested"`
}

type testNested struct {
	String string     `yaml:"string"`
	Bool   bool       `yaml:"bool"`
	Level  slog.Level `yaml:"level"`
	List   []string   `yaml:"list"`
}

func noLookup(string) (string, bool) {
	return "", false
}

func withEnv(env Env) Option {
	return WithResolver(schemeEnv, envResolver(env))
}

func lookupFrom(vars map[string]string) Env {
	return func(name string) (string, bool) {
		v, ok := vars[name]

		return v, ok
	}
}

func testdataFile(name string) Option {
	return WithPath(filepath.Join("testdata", name))
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	got, err := load[testConfig](t.Context(), testdataFile("valid.yml"), withEnv(noLookup))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	want := testConfig{
		Duration: 10 * time.Second,
		Nested: testNested{
			String: "value",
			Bool:   false,
			Level:  slog.LevelInfo,
			List:   []string{"a", "b"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("load() = %+v, want %+v", got, want)
	}
}

func TestLoadPrefersLookupOverDefaults(t *testing.T) {
	t.Parallel()

	lookup := lookupFrom(
		map[string]string{
			"DURATION":  "30s",
			"STRING":    "other",
			"BOOL":      "true",
			"LEVEL":     "debug",
			"LIST_ITEM": "x",
		},
	)

	got, err := load[testConfig](t.Context(), testdataFile("valid.yml"), withEnv(lookup))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	want := testConfig{
		Duration: 30 * time.Second,
		Nested: testNested{
			String: "other",
			Bool:   true,
			Level:  slog.LevelDebug,
			List:   []string{"x", "b"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("load() = %+v, want %+v", got, want)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		file    string
		wantErr error
	}{
		{name: "no such file", file: "absent.yml", wantErr: ErrReadFile},
		{name: "malformed yaml", file: "broken.yml", wantErr: ErrParseYAML},
		{name: "unknown key", file: "unknown.yml", wantErr: ErrDecodeConfig},
		{name: "missing references", file: "unresolved.yml", wantErr: ErrMissingReference},
		{name: "missing scheme", file: "noscheme.yml", wantErr: ErrInvalidReference},
		{name: "unknown scheme", file: "unknown_scheme.yml", wantErr: ErrInvalidReference},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := load[testConfig](t.Context(), testdataFile(tt.file), withEnv(noLookup))
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("load() error = %v, want %v", err, tt.wantErr)
				}
			},
		)
	}
}

func TestLoadReportsMissingReferences(t *testing.T) {
	t.Parallel()

	_, err := load[testConfig](t.Context(), testdataFile("unresolved.yml"), withEnv(noLookup))
	if err == nil {
		t.Fatal("load() error = nil, want missing references")
	}

	for _, ref := range []string{"${env:DURATION}", "${env:STRING}"} {
		if !strings.Contains(err.Error(), ref) {
			t.Errorf("load() error = %q, want it to mention %s", err, ref)
		}
	}
}

func TestLoadReadsEnvFile(t *testing.T) {
	t.Parallel()

	path := writeEnv(t, "STRING=from-file")

	got, err := load[testConfig](t.Context(), testdataFile("valid.yml"), WithEnvPath(path))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.Nested.String != "from-file" {
		t.Errorf("Nested.String = %q, want = %q", got.Nested.String, "from-file")
	}
}

func TestLoadValidates(t *testing.T) {
	t.Parallel()

	_, err := Load[validatable](t.Context(), testdataFile("empty.yml"), withEnv(noLookup))
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Load() error = %v, want %v", err, ErrValidation)
	}
}

func TestLoadUsesRegisteredResolver(t *testing.T) {
	t.Parallel()

	var gotRef string

	resolver := ResolverFunc(
		func(_ context.Context, ref string) (string, error) {
			gotRef = ref

			return "line1\nline2", nil
		},
	)

	got, err := load[testConfig](t.Context(), testdataFile("custom.yml"), WithResolver("custom", resolver))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if gotRef != "key" {
		t.Errorf("resolver got ref %q, want %q", gotRef, "key")
	}

	if got.Nested.String != "line1\nline2" {
		t.Errorf("Nested.String = %q, want %q", got.Nested.String, "line1\nline2")
	}
}

func TestLoadPassesContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	resolver := ResolverFunc(
		func(ctx context.Context, _ string) (string, error) {
			return "", ctx.Err()
		},
	)

	_, err := load[testConfig](ctx, testdataFile("custom.yml"), WithResolver("custom", resolver))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("load() error = %v, want %v", err, context.Canceled)
	}
}
