package server

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
)

const defaultShutdownTimeout = 5 * time.Second

type ServeOptions struct {
	ShutdownTimeout time.Duration
	BeforeStop      func()
}

func ListenAndServe(ctx context.Context, srv *grpc.Server, addr string, opts ServeOptions, log *slog.Logger) error {
	var lc net.ListenConfig

	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	return Serve(ctx, srv, lis, opts, log)
}

func Serve(ctx context.Context, srv *grpc.Server, lis net.Listener, opts ServeOptions, log *slog.Logger) error {
	log.InfoContext(
		ctx, "grpc server listening",
		slog.String("addr", lis.Addr().String()),
	)

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- srv.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)

	case <-ctx.Done():
	}

	if opts.BeforeStop != nil {
		opts.BeforeStop()
	}

	stop(ctx, srv, cmp.Or(opts.ShutdownTimeout, defaultShutdownTimeout), log)

	return <-serveErr
}

func stop(ctx context.Context, srv *grpc.Server, timeout time.Duration, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.InfoContext(ctx, "grpc server stopped gracefully")

	case <-ctx.Done():
		log.WarnContext(
			ctx, "grpc server did not drain in time, forcing stop",
			slog.Duration("timeout", timeout),
		)

		srv.Stop()
		<-done
	}
}
