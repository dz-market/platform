package config

import (
	"context"
	"errors"
	"reflect"
	"testing"
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
			schemeEnv: envResolver(testLookup),
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
		{name: "missing scheme", in: "${SET}", wantErr: ErrInvalidReference},
		{name: "unknown scheme", in: "${unknown:key}", wantErr: ErrInvalidReference},
		{name: "empty reference", in: "${}", wantErr: ErrInvalidReference},
		{name: "rejected by resolver", in: "${env:}", wantErr: ErrInvalidReference},
		{name: "missing value", in: "${env:UNSET}", wantErr: ErrMissingReference},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				e := newTestExpander()

				got := e.string(t.Context(), tt.in)
				if err := e.err(); !errors.Is(err, tt.wantErr) {
					t.Errorf("string(%q) error = %v, want %v", tt.in, err, tt.wantErr)
				}

				if got != tt.want {
					t.Errorf("string(%q) = %q, want %q", tt.in, got, tt.want)
				}
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
				if err := e.err(); !errors.Is(err, tt.wantErr) {
					t.Fatalf("node() error = %v, want %v", err, tt.wantErr)
				}

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("node() = %#v, want %#v", got, tt.want)
				}
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
				func(context.Context, string) (string, error) {
					calls++

					return "value", nil
				},
			),
		},
	)

	e.string(t.Context(), "${count:x}${count:x}")

	if calls != 1 {
		t.Errorf("resolver calls = %d, want 1", calls)
	}
}

func TestExpanderResolverFailure(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	e := newExpander(
		map[string]Resolver{
			"fail": ResolverFunc(
				func(context.Context, string) (string, error) {
					return "", boom
				},
			),
		},
	)

	e.string(t.Context(), "${fail:x}")

	err := e.err()
	if !errors.Is(err, ErrResolverFailed) {
		t.Errorf("err() = %v, want %v", err, ErrResolverFailed)
	}

	if !errors.Is(err, boom) {
		t.Errorf("err() = %v, want it to wrap %v", err, boom)
	}
}
