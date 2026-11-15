package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeEnv(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)

	return path
}

func TestEnvLookupPrecedence(t *testing.T) {
	path := writeEnv(t, "ONLY_FILE=file\nBOTH=file\n")

	t.Setenv("ONLY_ENV", "env")
	t.Setenv("BOTH", "env")

	env, err := LoadEnv(path)
	require.NoError(t, err)

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
				got, ok := env(tt.key)
				require.Equal(t, tt.want, got)
				require.Equal(t, tt.wantOK, ok)
			},
		)
	}
}

func TestEnvLookupMissingFileIsNotAnError(t *testing.T) {
	t.Parallel()

	env, err := LoadEnv(filepath.Join(t.TempDir(), "absent"))
	require.NoError(t, err)

	_, ok := env("ANYTHING")
	require.False(t, ok)
}

func TestEnvLookupUnreadableFile(t *testing.T) {
	t.Parallel()

	_, err := LoadEnv(t.TempDir())
	require.Error(t, err)
}

func TestEnvResolver(t *testing.T) {
	t.Parallel()

	r := lookupFrom(
		map[string]string{
			"SET":   "value",
			"EMPTY": "",
		},
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
				require.ErrorIs(t, err, tt.wantErr)

				if tt.wantErr != nil {
					return
				}

				require.Equal(t, tt.want, got)
			},
		)
	}
}

func TestEnvRequired(t *testing.T) {
	t.Parallel()

	env := lookupFrom(
		map[string]string{
			"STRING": "value",
			"EMPTY":  "",
		},
	)

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr error
	}{
		{name: "string", key: "STRING", want: "value"},
		{name: "missing", key: "MISSING", wantErr: ErrNotFound},
		{name: "empty", key: "EMPTY", wantErr: ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := env.Required(tt.key)
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.want, got)
			},
		)
	}
}
