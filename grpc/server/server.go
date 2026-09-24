package server

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/dz-market/platform/grpc/interceptor"
)

const defaultShutdownTimeout = 5 * time.Second

type Options struct {
	Addr                  string
	Reflection            bool
	MaxRecvMsgSize        int
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	ShutdownTimeout       time.Duration

	Validator protovalidate.Validator

	Unary []grpc.UnaryServerInterceptor
}

type Server struct {
	grpc            *grpc.Server
	health          *health.Server
	addr            string
	shutdownTimeout time.Duration
	log             *slog.Logger
}

func New(opts Options, log *slog.Logger) *Server {
	unary := make([]grpc.UnaryServerInterceptor, 0, len(opts.Unary)+4)
	unary = append(
		unary, interceptor.RequestID(),
		interceptor.Logging(log),
		interceptor.Recovery(log),
	)
	unary = append(unary, opts.Unary...)
	unary = append(unary, interceptor.Validate(opts.Validator))

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unary...),
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionAge:      opts.MaxConnectionAge,
				MaxConnectionAgeGrace: opts.MaxConnectionAgeGrace,
			},
		),
	}

	if opts.MaxRecvMsgSize > 0 {
		serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize))
	}

	srv := grpc.NewServer(serverOpts...)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)

	if opts.Reflection {
		reflection.Register(srv)

		log.Warn("grpc reflection is enabled")
	}

	return &Server{
		grpc:            srv,
		health:          healthSrv,
		addr:            opts.Addr,
		shutdownTimeout: cmp.Or(opts.ShutdownTimeout, defaultShutdownTimeout),
		log:             log,
	}
}

func (s *Server) Run(ctx context.Context) error {
	var lc net.ListenConfig

	lis, err := lc.Listen(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}

	return s.Serve(ctx, lis)
}

func (s *Server) Serve(ctx context.Context, lis net.Listener) error {
	s.log.InfoContext(
		ctx, "grpc server listening",
		slog.String("addr", lis.Addr().String()),
	)

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- s.grpc.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)

	case <-ctx.Done():
	}

	s.shutdown(ctx)

	return <-serveErr
}

//nolint:revive // flag-parameter: the signature matched health.Checker's onChange.
func (s *Server) SetServing(serving bool) {
	status := healthpb.HealthCheckResponse_NOT_SERVING

	if serving {
		status = healthpb.HealthCheckResponse_SERVING
	}

	s.health.SetServingStatus("", status)
}

func (s *Server) Registrar() grpc.ServiceRegistrar {
	return s.grpc
}

func (s *Server) shutdown(ctx context.Context) {
	s.health.Shutdown()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.log.InfoContext(ctx, "grpc server stopped gracefully")

	case <-ctx.Done():
		s.log.WarnContext(
			ctx, "grpc server did not drain in time, forcing stop",
			slog.Duration("timeout", s.shutdownTimeout),
		)

		s.grpc.Stop()
		<-done
	}
}
