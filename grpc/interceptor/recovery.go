package interceptor

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Recovery(log *slog.Logger, opts ...recovery.Option) grpc.UnaryServerInterceptor {
	options := make([]recovery.Option, 0, len(opts)+1)

	options = append(
		options,
		recovery.WithRecoveryHandlerContext(
			func(ctx context.Context, p any) (err error) {
				logger := log

				if id := RequestIDFromContext(ctx); id != "" {
					logger = logger.With(slog.String("request_id", id))
				}

				logger.ErrorContext(
					ctx, "panic recovered",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)

				return status.Error(codes.Internal, "internal error")
			},
		),
	)
	options = append(options, opts...)

	return recovery.UnaryServerInterceptor(options...)
}
