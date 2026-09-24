package interceptor

import (
	"log/slog"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
)

func Default(log *slog.Logger, validator protovalidate.Validator, inner ...grpc.UnaryServerInterceptor) []grpc.UnaryServerInterceptor {
	chain := make([]grpc.UnaryServerInterceptor, 0, len(inner)+4)
	chain = append(chain, RequestID(), Logging(log), Recovery(log))
	chain = append(chain, inner...)


	return append(chain, Validate(validator))
}
