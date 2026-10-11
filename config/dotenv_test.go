package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeEnv(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	return path
}

func TestEnvLookupPrecedence(t *testing.T) {
	path := writeEnv(t, "ONLY_FILE=file\nBOTH=file\n")

	t.Setenv("ONLY_ENV", "env")
	t.Setenv("BOTH", "env")

	lookup, err := envLookup(path)
	if err != nil {
		t.Fatalf("envLookup: %v", err)
	}

	tests := []struct {
		name   string
		key    string
		want   string
		wantOK bool
	}{
		{name: "only in file", key: "ONLY_FILE", want: "file", wantOK: true},
		{name: "only in environment", key: "ONLY_ENV", want: "env", wantOK: true},
		{name: "environment wins over file", key: "BOTH", want: "env", wantOK: true},
		{name: "nowhere", key: "ABSENT", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, ok := lookup(tt.key)
				if got != tt.want || ok != tt.wantOK {
					t.Errorf("lookup(%q) = (%q, %v), want (%q, %v)", tt.key, got, ok, tt.want, tt.wantOK)
				}
			},
		)
	}
}

func TestEnvLookupMissingFileIsNotAnError(t *testing.T) {
	t.Parallel()

	lookup, err := envLookup(filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("envLookup: %v", err)
	}

	if _, ok := lookup("ANYTHING"); ok {
		t.Error("lookup found a value with no .env file")
	}
}

func TestEnvLookupUnreadableFile(t *testing.T) {
	t.Parallel()

	if _, err := envLookup(t.TempDir()); err == nil {
		t.Error("envLookup on a directory: want error, got nil")
	}
}

func TestEnvResolver(t *testing.T) {
	t.Parallel()

	r := envResolver(
		lookupFrom(
			map[string]string{
				"SET":   "value",
				"EMPTY": "",
			},
		),
	)

	tests := []struct {
		name    string
		ref     string
		want    string
		wantErr error
	}{
		{name: "set", ref: "SET", want: "value"},
		{name: "set to empty", ref: "EMPTY", want: ""},
		{name: "not found", ref: "UNSET", wantErr: ErrNotFound},
		{name: "empty name", ref: "", wantErr: ErrInvalidReference},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := r.Resolve(t.Context(), tt.ref)
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Resolve(%q) error = %v, want %v", tt.ref, err, tt.wantErr)
				}

				if got != tt.want {
					t.Errorf("Resolve(%q) = %q, want %q", tt.ref, got, tt.want)
				}
			},
		)
	}
}
