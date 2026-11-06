package transaction

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey struct{}

type Manager struct {
	logger       *slog.Logger
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func New(logger *slog.Logger, pool *pgxpool.Pool, queryTimeout time.Duration) *Manager {
	return &Manager{
		logger:       logger.With("component", "postgres_transaction"),
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func FromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(contextKey{}).(pgx.Tx)
	return tx, ok
}

func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := FromContext(ctx); ok {
		return fn(ctx)
	}

	beginCtx, cancel := context.WithTimeout(ctx, m.queryTimeout)
	tx, err := m.pool.BeginTx(beginCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	cancel()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	ctx = context.WithValue(ctx, contextKey{}, tx)
	defer func() {
		if recovered := recover(); recovered != nil {
			m.rollback(ctx, tx)
			panic(recovered)
		}
		if err != nil {
			m.rollback(ctx, tx)
			return
		}

		commitCtx, commitCancel := context.WithTimeout(ctx, m.queryTimeout)
		defer commitCancel()
		if commitErr := tx.Commit(commitCtx); commitErr != nil {
			err = fmt.Errorf("commit transaction: %w", commitErr)
		}
	}()

	return fn(ctx)
}

func (m *Manager) rollback(ctx context.Context, tx pgx.Tx) {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.queryTimeout)
	defer cancel()
	if err := tx.Rollback(rollbackCtx); err != nil {
		m.logger.ErrorContext(rollbackCtx, "failed to rollback transaction", "error", err)
	}
}
