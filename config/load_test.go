package config

import (
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

func lookupFrom(vars map[string]string) lookupFunc {
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

	got, err := load[testConfig](testdataFile("valid.yml"), withLookup(noLookup))
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

	got, err := load[testConfig](testdataFile("valid.yml"), withLookup(lookup))
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
		{name: "no such file", file: "absent.yml", wantErr: ErrRead},
		{name: "malformed yaml", file: "broken.yml", wantErr: ErrParse},
		{name: "unknown key", file: "unknown.yml", wantErr: ErrDecode},
		{
			name:    "unresolved references",
			file:    "unresolved.yml",
			wantErr: ErrUnresolved,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := load[testConfig](testdataFile(tt.file), withLookup(noLookup))
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("load() error = %v, want %v", err, tt.wantErr)
				}
			},
		)
	}
}

func TestLoadReportsMissingReferences(t *testing.T) {
	t.Parallel()

	_, err := load[testConfig](testdataFile("unresolved.yml"), withLookup(noLookup))
	if !strings.Contains(err.Error(), "DURATION, STRING") {
		t.Errorf("load() error = %q, want it to list the missing names", err)
	}
}

func TestLoadReadsEnvFile(t *testing.T) {
	t.Parallel()

	path := writeEnv(t, "STRING=from-file")

	got, err := load[testConfig](testdataFile("valid.yml"), WithEnvPath(path))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.Nested.String != "from-file" {
		t.Errorf("Nested.String = %q, want = %q", got.Nested.String, "from-file")
	}
}

func TestLoadValidates(t *testing.T) {
	t.Parallel()

	_, err := Load[validatable](testdataFile("empty.yml"), withLookup(noLookup))
	if !errors.Is(err, ErrValidate) {
		t.Errorf("Load() error = %v, want %v", err, ErrValidate)
	}
}
