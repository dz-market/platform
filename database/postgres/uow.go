package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultRollbackTimeout = 5 * time.Second

type UnitOfWorkOptions struct {
	TxOptions       pgx.TxOptions
	RollbackTimeout time.Duration
}

type UnitOfWork[R any] struct {
	pool  *pgxpool.Pool
	repos func(Querier) R
	opts  UnitOfWorkOptions
}

func NewUnitOfWork[R any](pool *pgxpool.Pool, opts UnitOfWorkOptions, repos func(Querier) R) (*UnitOfWork[R], error) {
	if opts.RollbackTimeout < 0 {
		return nil, errors.New("rollback timeout must not be negative")
	}

	if opts.RollbackTimeout == 0 {
		opts.RollbackTimeout = defaultRollbackTimeout
	}

	return &UnitOfWork[R]{
		pool:  pool,
		repos: repos,
		opts:  opts,
	}, nil
}

func (u *UnitOfWork[R]) Do(ctx context.Context, fn func(R) error) error {
	tx, err := u.pool.BeginTx(ctx, u.opts.TxOptions)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), u.opts.RollbackTimeout)
		defer cancel()

		_ = tx.Rollback(rollbackCtx)
	}()

	if err := fn(u.repos(tx)); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
