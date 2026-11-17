package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultInitialPingTimeout = 5 * time.Second

type Options struct {
	DSN string

	MaxConns     *int32
	MinConns     *int32
	MinIdleConns *int32

	MaxConnLifetime       *time.Duration
	MaxConnLifetimeJitter *time.Duration
	MaxConnIdleTime       *time.Duration
	HealthCheckPeriod     *time.Duration
	ConnectTimeout        *time.Duration

	ApplicationName *string

	InitialPingTimeout *time.Duration

	OverrideConfig func(*pgxpool.Config) error
}

func New(ctx context.Context, opts Options) (*pgxpool.Pool, error) {
	if opts.DSN == "" {
		return nil, errors.New("dsn is required")
	}

	initialPingTimeout := defaultInitialPingTimeout
	applyOption(&initialPingTimeout, opts.InitialPingTimeout)

	if initialPingTimeout <= 0 {
		return nil, errors.New("initial ping timeout must be positive")
	}

	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := applyOptions(cfg, opts); err != nil {
		return nil, err
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg.Copy())
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, initialPingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping pool: %w", err)
	}

	return pool, nil
}

func applyOptions(cfg *pgxpool.Config, opts Options) error {
	applyOption(&cfg.MaxConns, opts.MaxConns)
	applyOption(&cfg.MinConns, opts.MinConns)
	applyOption(&cfg.MinIdleConns, opts.MinIdleConns)
	applyOption(&cfg.MaxConnLifetime, opts.MaxConnLifetime)
	applyOption(&cfg.MaxConnLifetimeJitter, opts.MaxConnLifetimeJitter)
	applyOption(&cfg.MaxConnIdleTime, opts.MaxConnIdleTime)
	applyOption(&cfg.HealthCheckPeriod, opts.HealthCheckPeriod)
	applyOption(&cfg.ConnConfig.ConnectTimeout, opts.ConnectTimeout)

	if opts.ApplicationName != nil {
		cfg.ConnConfig.RuntimeParams["application_name"] = *opts.ApplicationName
	}

	if opts.OverrideConfig != nil {
		if err := opts.OverrideConfig(cfg); err != nil {
			return fmt.Errorf("override config: %w", err)
		}
	}

	return nil
}

func validateConfig(cfg *pgxpool.Config) error {
	if cfg.ConnConfig == nil {
		return errors.New("connection config is required")
	}

	if cfg.MaxConns <= 0 {
		return errors.New("max connections must be positive")
	}

	if cfg.MinConns < 0 || cfg.MinConns > cfg.MaxConns {
		return errors.New("min connections must be between zero and max connections")
	}

	if cfg.MinIdleConns < 0 || cfg.MinIdleConns > cfg.MaxConns {
		return errors.New("min idle connections must be between zero and max connections")
	}

	if cfg.HealthCheckPeriod <= 0 {
		return errors.New("health check period must be positive")
	}

	if cfg.MaxConnLifetime < 0 {
		return errors.New("max connection lifetime durations must not be negative")
	}

	if cfg.MaxConnLifetimeJitter < 0 {
		return errors.New("max connection lifetime jitter durations must not be negative")
	}

	if cfg.MaxConnIdleTime < 0 {
		return errors.New("max connection idle time must not be negative")
	}

	if cfg.ConnConfig.ConnectTimeout < 0 {
		return errors.New("connect timeout must not be negative")
	}

	return nil
}

func applyOption[T any](dst, value *T) {
	if value != nil {
		*dst = *value
	}
}
