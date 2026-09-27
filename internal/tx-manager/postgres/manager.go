package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/a-pelikh/tripgo/internal/tx-manager"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type manager struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func New(logger *slog.Logger, pool *pgxpool.Pool) *manager {
	return &manager{
		logger: logger.With("tx_manager", "postgres"),
		pool:   pool,
	}
}

func (m *manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(tx_manager.TxKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	ctx = context.WithValue(ctx, tx_manager.TxKey{}, tx)
	defer func() {
		if p := recover(); p != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				m.logger.ErrorContext(ctx, "failed to rollback transaction", "error", errRollback)
			}
			panic(p)
		} else if err != nil {
			if errRollback := tx.Rollback(ctx); errRollback != nil {
				m.logger.ErrorContext(ctx, "failed to rollback transaction", "error", errRollback)
			}
		} else {
			err = tx.Commit(ctx)
		}
	}()

	return fn(ctx)
}
