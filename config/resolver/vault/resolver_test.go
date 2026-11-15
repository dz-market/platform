package vault_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"

	"github.com/dz-market/platform/config"
	pvault "github.com/dz-market/platform/vault"

	"github.com/dz-market/platform/config/resolver/vault"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	const response = `{
		"data": {
			"string": "plain",
			"nested": {"string": "value"},
			"empty": "",
			"null": null,
			"number": 10,
			"boolean": true,
			"object": {},
			"list": ["item"]
		}
	}`

	tests := []struct {
		name    string
		ref     string
		status  int
		body    string
		want    any
		wantErr error
	}{
		{
			name: "string",
			ref:  "secret/example#string",
			want: "plain",
		},
		{
			name: "nested",
			ref:  "secret/example#nested.string",
			want: "value",
		},
		{
			name: "empty",
			ref:  "secret/example#empty",
			want: "",
		},
		{
			name: "number",
			ref:  "secret/example#number",
			want: json.Number("10"),
		},
		{
			name: "boolean",
			ref:  "secret/example#boolean",
			want: true,
		},
		{
			name: "null",
			ref:  "secret/example#null",
			want: nil,
		},
		{
			name: "object",
			ref:  "secret/example#object",
			want: map[string]any{},
		},
		{
			name: "list",
			ref:  "secret/example#list",
			want: []any{"item"},
		},
		{
			name:    "missing field",
			ref:     "secret/example#missing",
			wantErr: config.ErrNotFound,
		},
		{
			name:    "missing nested field",
			ref:     "secret/example#nested.missing",
			wantErr: config.ErrNotFound,
		},
		{
			name:    "non-object parent",
			ref:     "secret/example#string.child",
			wantErr: config.ErrInvalidReference,
		},
		{
			name:    "missing separator",
			ref:     "secret/example",
			wantErr: config.ErrInvalidReference,
		},
		{
			name:    "missing path",
			ref:     "#string",
			wantErr: config.ErrInvalidReference,
		},
		{
			name:    "missing field reference",
			ref:     "secret/example#",
			wantErr: config.ErrInvalidReference,
		},
		{
			name:    "Vault path not found",
			ref:     "secret/example#string",
			status:  http.StatusNotFound,
			body:    `{}`,
			wantErr: config.ErrNotFound,
		},
		{
			name:    "Vault response without data",
			ref:     "secret/example#string",
			body:    `{}`,
			wantErr: pvault.ErrNoData,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				status := tt.status
				if status == 0 {
					status = http.StatusOK
				}

				body := tt.body
				if body == "" {
					body = response
				}

				resolver := newTestResolver(t, status, body)

				got, err := resolver.Resolve(t.Context(), tt.ref)
				require.ErrorIs(t, err, tt.wantErr)

				if tt.wantErr != nil {
					require.Nil(t, got)

					return
				}

				require.Equal(t, tt.want, got)
			},
		)
	}
}

func TestResolveReadError(t *testing.T) {
	t.Parallel()

	resolver := newTestResolver(t, http.StatusForbidden, `{}`)

	got, err := resolver.Resolve(t.Context(), "secret/example#string")
	require.Error(t, err)
	require.Nil(t, got)

	var responseErr *api.ResponseError
	require.ErrorAs(t, err, &responseErr)
	require.Equal(t, http.StatusForbidden, responseErr.StatusCode)
}

func newTestResolver(t *testing.T, status int, body string) *vault.Resolver {
	t.Helper()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)

				_, _ = w.Write([]byte(body))
			},
		),
	)
	t.Cleanup(server.Close)

	cfg := api.DefaultConfig()
	cfg.Address = server.URL
	cfg.HttpClient = server.Client()
	cfg.MaxRetries = 0

	client, err := pvault.New(cfg)
	require.NoError(t, err)

	return vault.New(client)
}
