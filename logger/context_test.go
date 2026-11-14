package logger_test

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"testing"

	"github.com/dz-market/platform/logger"
)

const userID = "user-1"

func TestContextAttributes(t *testing.T) {
	parent := logger.With(t.Context(), slog.String("user_id", userID))
	child := logger.With(parent, slog.String("request_id", "request-1"))

	tests := []struct {
		name          string
		child         bool
		wantRequestID string
	}{
		{
			name: "parent",
		},
		{
			name:          "child",
			child:         true,
			wantRequestID: "request-1",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				var output bytes.Buffer

				log := slog.New(
					logger.ContextHandler(
						slog.NewJSONHandler(&output, nil),
					),
				)

				ctx := parent
				if tt.child {
					ctx = child
				}

				log.InfoContext(ctx, "test")

				var record struct {
					UserID    string `json:"user_id"`
					RequestID string `json:"request_id"`
				}

				if err := json.Unmarshal(output.Bytes(), &record); err != nil {
					t.Fatalf("decode log: %v", err)
				}

				if record.UserID != userID {
					t.Errorf("user ID = %q, want %q", record.UserID, userID)
				}

				if record.RequestID != tt.wantRequestID {
					t.Errorf("request ID = %q, want %q", record.RequestID, tt.wantRequestID)
				}
			},
		)
	}
}
