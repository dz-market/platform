package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func envFrom(vars map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		v, ok := vars[name]

		return v, ok
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	return path
}

func TestConfigFromEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vars    map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "values",
			vars: map[string]string{
				"VAULT_ADDR":      "http://vault:8200",
				"VAULT_ROLE_ID":   "role",
				"VAULT_SECRET_ID": "secret",
			},
			want: Config{
				Addr:     "http://vault:8200",
				RoleID:   "role",
				SecretID: "secret",
			},
		},
		{
			name: "files",
			vars: map[string]string{
				"VAULT_ADDR":           "http://vault:8200",
				"VAULT_ROLE_ID_FILE":   writeFile(t, "role"),
				"VAULT_SECRET_ID_FILE": writeFile(t, "secret"),
			},
			want: Config{
				Addr:     "http://vault:8200",
				RoleID:   "role",
				SecretID: "secret",
			},
		},
		{
			name: "value wins over file",
			vars: map[string]string{
				"VAULT_ROLE_ID":      "role",
				"VAULT_ROLE_ID_FILE": writeFile(t, "from-file"),
			},
			want: Config{
				RoleID: "role",
			},
		},
		{
			name: "nothing set",
			vars: map[string]string{},
			want: Config{},
		},
		{
			name: "unreadable file",
			vars: map[string]string{
				"VAULT_SECRET_ID_FILE": filepath.Join(t.TempDir(), "unreadable"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := ConfigFromEnv(envFrom(tt.vars))
				if (err != nil) != tt.wantErr {
					t.Errorf("ConfigFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				}

				if got != tt.want {
					t.Errorf("ConfigFromEnv() got = %+v, want %+v", got, tt.want)
				}
			},
		)
	}
}
