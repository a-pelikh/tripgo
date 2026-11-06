package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/a-pelikh/tripgo/internal/tx-manager"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type manager struct {
	logger       *slog.Logger
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func New(logger *slog.Logger, pool *pgxpool.Pool, queryTimeout time.Duration) *manager {
	return &manager{
		logger:       logger.With("tx_manager", "postgres"),
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (m *manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(tx_manager.TxKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	beginCtx, beginCancel := context.WithTimeout(ctx, m.queryTimeout)
	tx, err := m.pool.BeginTx(beginCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	beginCancel()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	ctx = context.WithValue(ctx, tx_manager.TxKey{}, tx)
	defer func() {
		if p := recover(); p != nil {
			m.rollback(ctx, tx)
			panic(p)
		} else if err != nil {
			m.rollback(ctx, tx)
		} else {
			commitCtx, commitCancel := context.WithTimeout(ctx, m.queryTimeout)
			defer commitCancel()
			if commitErr := tx.Commit(commitCtx); commitErr != nil {
				err = fmt.Errorf("commit transaction: %w", commitErr)
			}
		}
	}()

	return fn(ctx)
}

func (m *manager) rollback(ctx context.Context, tx pgx.Tx) {
	rollbackCtx, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), m.queryTimeout)
	defer rollbackCancel()

	if err := tx.Rollback(rollbackCtx); err != nil {
		m.logger.ErrorContext(rollbackCtx, "failed to rollback transaction", "error", err)
	}
}
