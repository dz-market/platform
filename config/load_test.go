package config

import (
	"context"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	return WithResolver(schemeEnv, env)
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
	require.NoError(t, err)

	want := testConfig{
		Duration: 10 * time.Second,
		Nested: testNested{
			String: "value",
			Bool:   false,
			Level:  slog.LevelInfo,
			List:   []string{"a", "b"},
		},
	}

	require.Equal(t, want, got)
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
	require.NoError(t, err)

	want := testConfig{
		Duration: 30 * time.Second,
		Nested: testNested{
			String: "other",
			Bool:   true,
			Level:  slog.LevelDebug,
			List:   []string{"x", "b"},
		},
	}

	require.Equal(t, want, got)
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
				require.ErrorIs(t, err, tt.wantErr)
			},
		)
	}
}

func TestLoadReportsMissingReferences(t *testing.T) {
	t.Parallel()

	_, err := load[testConfig](t.Context(), testdataFile("unresolved.yml"), withEnv(noLookup))
	require.Error(t, err)

	for _, ref := range []string{"${env:DURATION}", "${env:STRING}"} {
		require.ErrorContains(t, err, ref)
	}
}

func TestLoadReadsEnvFile(t *testing.T) {
	t.Parallel()

	path := writeEnv(t, "STRING=from-file")

	got, err := load[testConfig](t.Context(), testdataFile("valid.yml"), WithEnvPath(path))
	require.NoError(t, err)
	require.Equal(t, "from-file", got.Nested.String)
}

func TestLoadValidates(t *testing.T) {
	t.Parallel()

	_, err := Load[validatable](t.Context(), testdataFile("empty.yml"), withEnv(noLookup))
	require.ErrorIs(t, err, ErrValidation)
}

func TestLoadUsesRegisteredResolver(t *testing.T) {
	t.Parallel()

	var gotRef string

	resolver := ResolverFunc(
		func(_ context.Context, ref string) (any, error) {
			gotRef = ref

			return "line1\nline2", nil
		},
	)

	got, err := load[testConfig](
		t.Context(),
		testdataFile("custom.yml"),
		WithResolver("custom", resolver),
	)
	require.NoError(t, err)
	require.Equal(t, "key", gotRef)
	require.Equal(t, "line1\nline2", got.Nested.String)
}

func TestLoadTypedValues(t *testing.T) {
	t.Parallel()

	type objectValue struct {
		String string `yaml:"string"`
		Number int    `yaml:"number"`
	}

	type valueConfig struct {
		Object  objectValue `yaml:"object"`
		List    []string    `yaml:"list"`
		Number  int         `yaml:"number"`
		Boolean bool        `yaml:"boolean"`
		Null    *string     `yaml:"null"`
		Text    string      `yaml:"text"`
	}

	values := map[string]any{
		"object": map[string]any{
			"string": "value",
			"number": json.Number("10"),
		},
		"list":    []any{"value"},
		"number":  json.Number("10"),
		"boolean": true,
		"null":    nil,
	}

	resolver := ResolverFunc(
		func(_ context.Context, ref string) (any, error) {
			value, ok := values[ref]
			if !ok {
				return nil, ErrNotFound
			}

			return value, nil
		},
	)

	got, err := Load[valueConfig](
		t.Context(),
		testdataFile("typed.yml"),
		withEnv(noLookup),
		WithResolver("test", resolver),
	)
	require.NoError(t, err)

	want := valueConfig{
		Object: objectValue{
			String: "value",
			Number: 10,
		},
		List:    []string{"value"},
		Number:  10,
		Boolean: true,
		Text:    "boolean=true",
	}

	require.Equal(t, want, got)
}

func TestLoadPassesContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	resolver := ResolverFunc(
		func(ctx context.Context, _ string) (any, error) {
			return nil, ctx.Err()
		},
	)

	_, err := load[testConfig](ctx, testdataFile("custom.yml"), WithResolver("custom", resolver))
	require.ErrorIs(t, err, context.Canceled)
}
