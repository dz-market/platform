package config

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func testLookup(name string) (string, bool) {
	v, ok := map[string]string{
		"SET":   "value",
		"EMPTY": "",
	}[name]

	return v, ok
}

func newTestExpander() *expander {
	return newExpander(
		map[string]Resolver{
			schemeEnv: Env(testLookup),
		},
	)
}

func TestExpanderString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{name: "no references", in: "plain text", want: "plain text"},
		{name: "resolved", in: "${env:SET}", want: "value"},
		{name: "resolved to empty value", in: "${env:EMPTY}", want: ""},
		{name: "inside text", in: "host=${env:SET}", want: "host=value"},
		{name: "repeated", in: "${env:SET}/${env:SET}", want: "value/value"},
		{name: "escaped reference", in: "$${env:SET}", want: "${env:SET}"},
		{name: "default applied", in: "${env:UNSET:-fallback}", want: "fallback"},
		{name: "default ignored when set", in: "${env:SET:-fallback}", want: "value"},
		{name: "default ignored when set to empty", in: "${env:EMPTY:-fallback}", want: ""},
		{name: "empty default", in: "${env:UNSET:-}", want: ""},
		{name: "default with colon", in: "${env:UNSET:-a:b}", want: "a:b"},
		{name: "missing scheme", in: "${SET}", wantErr: ErrInvalidReference},
		{name: "unknown scheme", in: "${unknown:key}", wantErr: ErrInvalidReference},
		{name: "empty reference", in: "${}", wantErr: ErrInvalidReference},
		{name: "rejected by resolver", in: "${env:}", wantErr: ErrInvalidReference},
		{name: "default does not hide invalid reference", in: "${env::-key}", wantErr: ErrInvalidReference},
		{name: "missing value", in: "${env:UNSET}", wantErr: ErrMissingReference},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				e := newTestExpander()

				got := e.string(t.Context(), tt.in)
				require.ErrorIs(t, e.err(), tt.wantErr)
				require.Equal(t, tt.want, got)
			},
		)
	}
}

func TestExpanderTypedValues(t *testing.T) {
	t.Parallel()

	object := map[string]any{"string": "value"}
	list := []any{"value"}
	number := json.Number("10")

	tests := []struct {
		name    string
		in      string
		value   any
		want    any
		wantErr error
	}{
		{
			name:  "object",
			in:    "${test:value}",
			value: object,
			want:  object,
		},
		{
			name:  "list",
			in:    "${test:value}",
			value: list,
			want:  list,
		},
		{
			name:  "number",
			in:    "${test:value}",
			value: number,
			want:  number,
		},
		{
			name:  "boolean",
			in:    "${test:value}",
			value: true,
			want:  true,
		},
		{
			name: "null",
			in:   "${test:value}",
		},
		{
			name: "null does not activate default",
			in:   "${test:value:-fallback}",
		},
		{
			name:  "number inside text",
			in:    "number=${test:value}",
			value: number,
			want:  "number=10",
		},
		{
			name:  "boolean inside text",
			in:    "boolean=${test:value}",
			value: true,
			want:  "boolean=true",
		},
		{
			name:    "object inside text",
			in:      "object=${test:value}",
			value:   object,
			wantErr: ErrInvalidValue,
		},
		{
			name:    "list inside text",
			in:      "list=${test:value}",
			value:   list,
			wantErr: ErrInvalidValue,
		},
		{
			name:    "null inside text",
			in:      "null=${test:value}",
			wantErr: ErrInvalidValue,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				resolver := ResolverFunc(
					func(context.Context, string) (any, error) {
						return tt.value, nil
					},
				)

				e := newExpander(
					map[string]Resolver{
						"test": resolver,
					},
				)

				got := e.string(t.Context(), tt.in)
				require.ErrorIs(t, e.err(), tt.wantErr)

				if tt.wantErr != nil {
					return
				}

				require.Equal(t, tt.want, got)
			},
		)
	}
}

func TestExpanderNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		node    any
		want    any
		wantErr error
	}{
		{name: "string is expanded", node: "${env:SET}", want: "value"},
		{name: "number is left alone", node: 10, want: 10},
		{name: "bool is left alone", node: true, want: true},
		{
			name: "map is expanded recursively",
			node: map[string]any{"key": "${env:SET}"},
			want: map[string]any{"key": "value"},
		},
		{
			name: "list is expanded element by element",
			node: []any{"${env:SET}", "plain"},
			want: []any{"value", "plain"},
		},
		{
			name:    "missing value deep inside is reported",
			node:    map[string]any{"outer": map[string]any{"inner": "${env:UNSET}"}},
			want:    map[string]any{"outer": map[string]any{"inner": ""}},
			wantErr: ErrMissingReference,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				e := newTestExpander()

				got := e.node(t.Context(), tt.node)
				require.ErrorIs(t, e.err(), tt.wantErr)
				require.Equal(t, tt.want, got)
			},
		)
	}
}

func TestExpanderCachesResults(t *testing.T) {
	t.Parallel()

	calls := 0
	e := newExpander(
		map[string]Resolver{
			"count": ResolverFunc(
				func(context.Context, string) (any, error) {
					calls++

					return "value", nil
				},
			),
		},
	)

	e.string(t.Context(), "${count:x}${count:x:-fallback}")

	require.Equal(t, 1, calls)
}

func TestExpanderResolverFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	e := newExpander(
		map[string]Resolver{
			"fail": ResolverFunc(
				func(context.Context, string) (any, error) {
					return nil, boom
				},
			),
		},
	)

	e.string(t.Context(), "${fail:key:-fallback}")

	err := e.err()
	require.ErrorIs(t, err, ErrResolverFailed)
	require.ErrorIs(t, err, boom)
}
