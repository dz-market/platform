package interceptor

import (
	"context"
	"log/slog"
	"uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/dz-market/platform/logger"
)

const HeaderRequestID = "x-request-id"

const maxRequestIDLen = 128

func RequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		id := incomingRequestID(ctx)

		ctx = logger.With(ctx, slog.String("request_id", id))

		_ = grpc.SetHeader(ctx, metadata.Pairs(HeaderRequestID, id))

		return handler(ctx, req)
	}
}

func incomingRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(HeaderRequestID); len(values) > 0 && values[0] != "" && len(values[0]) <= maxRequestIDLen {
			return values[0]
		}
	}

	return uuid.NewV4().String()
}
