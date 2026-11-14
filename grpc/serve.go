package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
)

const defaultShutdownTimeout = 5 * time.Second

type ServeOptions struct {
	ShutdownTimeout time.Duration
	BeforeStop      func()
}

func ListenAndServe(ctx context.Context, srv *grpc.Server, addr string, opts ServeOptions) error {
	var lc net.ListenConfig

	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	return Serve(ctx, srv, lis, opts)
}

func Serve(ctx context.Context, srv *grpc.Server, lis net.Listener, opts ServeOptions) error {
	defer lis.Close()

	if opts.ShutdownTimeout < 0 {
		return errors.New("shutdown timeout must not be negative")
	}

	if ctx.Err() != nil {
		return nil
	}

	timeout := opts.ShutdownTimeout
	if timeout == 0 {
		timeout = defaultShutdownTimeout
	}

	done := make(chan error, 1)

	go func() {
		done <- srv.Serve(lis)
	}()

	select {
	case err := <-done:
		srv.Stop()

		return err

	case <-ctx.Done():
	}

	if opts.BeforeStop != nil {
		opts.BeforeStop()
	}

	shutdown(srv, timeout)

	err := <-done
	if errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}

	return err
}

func shutdown(srv *grpc.Server, timeout time.Duration) {
	done := make(chan struct{})

	go func() {
		srv.GracefulStop()
		close(done)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		srv.Stop()
		<-done
	}
}
