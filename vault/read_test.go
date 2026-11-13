package vault_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/vault/api"

	"github.com/dz-market/platform/vault"
)

func TestReadErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{
			name:    "missing path",
			status:  http.StatusNotFound,
			wantErr: vault.ErrPathNotFound,
		},
		{
			name:    "response without data",
			status:  http.StatusOK,
			body:    `{}`,
			wantErr: vault.ErrNoData,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				server := httptest.NewServer(
					http.HandlerFunc(
						func(w http.ResponseWriter, _ *http.Request) {
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(tt.status)
							_, _ = io.WriteString(w, tt.body)
						},
					),
				)
				defer server.Close()

				client, err := vault.New(
					&api.Config{
						Address:    server.URL,
						HttpClient: server.Client(),
					},
				)
				if err != nil {
					t.Fatalf("create client: %v", err)
				}

				_, err = client.Read(t.Context(), "secret/example")
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Read() error = %v, want %v", err, tt.wantErr)
				}
			},
		)
	}
}
