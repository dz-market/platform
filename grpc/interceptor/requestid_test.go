package interceptor_test

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/dz-market/platform/grpc/interceptor"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		keep bool
	}{
		{name: "valid", id: "request-123", keep: true},
		{name: "max length", id: strings.Repeat("a", 128), keep: true},
		{name: "missing", id: "", keep: false},
		{name: "too long", id: strings.Repeat("a", 129), keep: false},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs(interceptor.HeaderRequestID, tt.id))

				_, err := interceptor.RequestID()(
					ctx, nil, nil, func(ctx context.Context, _ any) (any, error) {
						id := interceptor.RequestIDFromContext(ctx)

						if tt.keep && id != tt.id {
							t.Errorf("ID = %q, want = %q", id, tt.id)
						}
						if !tt.keep && (id == "" || id == tt.id) {
							t.Errorf("expected generated ID, got %q", id)
						}

						return nil, nil
					},
				)
				if err != nil {
					t.Fatal(err)
				}
			},
		)
	}
}
