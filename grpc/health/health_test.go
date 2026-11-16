package health_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/dz-market/platform/grpc/health"
)

func TestChecker_HealthTransitions(t *testing.T) {
	synctest.Test(
		t, func(t *testing.T) {
			const period = time.Second

			checker, registrar := newChecker(
				t, health.Options{
					Period:  period,
					Timeout: time.Second,
				},
			)

			unavailable := false

			require.NoError(
				t, checker.Register(
					"dep-1", health.DependencyOptions{}, func(context.Context) error {
						return nil
					},
				),
			)

			require.NoError(
				t, checker.Register(
					"dep-2", health.DependencyOptions{}, func(context.Context) error {
						if unavailable {
							return errors.New("dep-2 unavailable")
						}

						return nil
					},
				),
			)

			assertStatus(t, registrar, healthpb.HealthCheckResponse_NOT_SERVING)

			startChecker(t, checker)
			assertStatus(t, registrar, healthpb.HealthCheckResponse_SERVING)

			unavailable = true

			time.Sleep(period)
			synctest.Wait()

			assertStatus(t, registrar, healthpb.HealthCheckResponse_NOT_SERVING)

			unavailable = false

			time.Sleep(period)
			synctest.Wait()

			assertStatus(t, registrar, healthpb.HealthCheckResponse_SERVING)
		},
	)
}

func TestChecker_DependencyTimeout(t *testing.T) {
	tests := []struct {
		name     string
		override time.Duration
		want     time.Duration
	}{
		{
			name: "default",
			want: 5 * time.Second,
		},
		{
			name:     "override",
			override: 2 * time.Second,
			want:     2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				synctest.Test(
					t, func(t *testing.T) {
						checker, _ := newChecker(
							t, health.Options{
								Period:  time.Minute,
								Timeout: 5 * time.Second,
							},
						)

						var gotTimeout time.Duration

						require.NoError(
							t, checker.Register(
								"dep",
								health.DependencyOptions{Timeout: tt.override},
								func(ctx context.Context) error {
									if deadline, ok := ctx.Deadline(); ok {
										gotTimeout = time.Until(deadline)
									}

									return nil
								},
							),
						)

						startChecker(t, checker)

						require.Equal(t, tt.want, gotTimeout)
					},
				)
			},
		)
	}
}

type healthRegistrar struct {
	server healthpb.HealthServer
}

func (r *healthRegistrar) RegisterService(_ *grpc.ServiceDesc, impl any) {
	server, ok := impl.(healthpb.HealthServer)
	if !ok {
		return
	}

	r.server = server
}

func newChecker(t *testing.T, opts health.Options) (*health.Checker, *healthRegistrar) {
	t.Helper()

	registrar := &healthRegistrar{}
	checker, err := health.New(registrar, opts)
	require.NoError(t, err)
	require.NotNil(t, registrar.server)

	return checker, registrar
}

func assertStatus(t *testing.T, registrar *healthRegistrar, want healthpb.HealthCheckResponse_ServingStatus) {
	t.Helper()

	response, err := registrar.server.Check(t.Context(), &healthpb.HealthCheckRequest{Service: ""})
	require.NoError(t, err)
	require.Equal(t, want, response.GetStatus())
}

func startChecker(t *testing.T, checker *health.Checker) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		done <- checker.Run(ctx)
	}()

	t.Cleanup(
		func() {
			cancel()
			synctest.Wait()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Second)
			defer shutdownCancel()

			require.NoError(t, checker.Shutdown(shutdownCtx))
			require.NoError(t, <-done)
		},
	)

	synctest.Wait()
}
