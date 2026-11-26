package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/dz-market/platform/logctx"

	"github.com/dz-market/platform/logger"
)

const requestID = "request-1"

func TestNewIncludesContextAttributes(t *testing.T) {
	var output bytes.Buffer

	log, err := logger.New(
		logger.Options{
			Output: &output,
		},
	)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	ctx := logctx.With(t.Context(), slog.String("request_id", requestID))
	log.InfoContext(ctx, "handler")

	var record struct {
		RequestID string `json:"request_id"`
	}

	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}

	if record.RequestID != requestID {
		t.Errorf("request ID = %q, want %q", record.RequestID, requestID)
	}
}
