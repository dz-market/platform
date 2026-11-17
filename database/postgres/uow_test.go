//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dz-market/platform/database/postgres"
)

const (
	postgresImage    = "postgres:18-alpine"
	startupTimeout   = time.Minute
	operationTimeout = 10 * time.Second

	insertQuery = "INSERT INTO entries (id) VALUES ($1)"
)

func TestUnitOfWork(t *testing.T) {
	dsn := startPostgres(t)

	callbackErr := errors.New("operation failed")
	panicValue := errors.New("callback panic")

	tests := []struct {
		name        string
		callbackErr error
		cancel      bool
		panicValue  any
		wantErr     error
		wantPanic   any
		wantCount   int
	}{
		{
			name:      "commit",
			wantCount: 1,
		},
		{
			name:        "rollback",
			callbackErr: callbackErr,
			wantErr:     callbackErr,
		},
		{
			name:       "panic",
			panicValue: panicValue,
			wantPanic:  panicValue,
		},
		{
			name:    "cancellation",
			cancel:  true,
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				pool, uow := newUnitOfWork(t, dsn)

				ctx, cancel := context.WithTimeout(t.Context(), operationTimeout)
				defer cancel()

				var gotErr error
				var gotPanic any

				func() {
					defer func() {
						gotPanic = recover()
					}()

					gotErr = uow.Do(
						ctx, func(q postgres.Querier) error {
							_, err := q.Exec(ctx, insertQuery, 1)
							if err != nil {
								return err
							}

							if tt.panicValue != nil {
								panic(tt.panicValue)
							}

							if tt.cancel {
								cancel()
							}

							return tt.callbackErr
						},
					)
				}()

				if !errors.Is(gotErr, tt.wantErr) {
					t.Fatalf("error = %v, want %v", gotErr, tt.wantErr)
				}

				if gotPanic != tt.wantPanic {
					t.Fatalf("panic = %v, want %v", gotPanic, tt.wantPanic)
				}

				checkRows(t, pool, tt.wantCount)
			},
		)
	}
}

func startPostgres(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), startupTimeout)
	defer cancel()

	container, err := tcpostgres.Run(
		ctx,
		postgresImage,
		tcpostgres.WithDatabase("platform_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)

	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get postgres DSN: %v", err)
	}

	return dsn
}

func newUnitOfWork(t *testing.T, dsn string) (*pgxpool.Pool, *postgres.UnitOfWork[postgres.Querier]) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), operationTimeout)
	defer cancel()

	pool, err := postgres.New(
		ctx, postgres.Options{
			DSN:      dsn,
			MaxConns: new(int32(1)),
			MinConns: new(int32(0)),
		},
	)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}

	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, "CREATE TEMP TABLE entries (id integer PRIMARY KEY)")
	if err != nil {
		t.Fatalf("create test table: %v", err)
	}

	uow, err := postgres.NewUnitOfWork(
		pool, postgres.UnitOfWorkOptions{}, func(q postgres.Querier) postgres.Querier {
			return q
		},
	)
	if err != nil {
		t.Fatalf("create uow: %v", err)
	}

	return pool, uow
}

func checkRows(t *testing.T, pool *pgxpool.Pool, want int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), operationTimeout)
	defer cancel()

	var count int

	err := pool.QueryRow(ctx, "SELECT count(*) FROM entries").Scan(&count)
	if err != nil {
		t.Fatalf("query after transaction: %v", err)
	}

	if count != want {
		t.Errorf("row count = %d, want %d", count, want)
	}
}
