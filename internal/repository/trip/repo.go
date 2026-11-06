package trip

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/a-pelikh/tripgo/internal/infra/postgres/transaction"
)

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func New(pool *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	return &Repository{pool: pool, queryTimeout: queryTimeout}
}

func (r *Repository) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	return r.pool.Ping(ctx)
}

func (r *Repository) queryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if tx, ok := transaction.FromContext(ctx); ok {
		return tx.QueryRow(ctx, query, args...)
	}
	return r.pool.QueryRow(ctx, query, args...)
}

func (r *Repository) exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := transaction.FromContext(ctx); ok {
		return tx.Exec(ctx, query, args...)
	}
	return r.pool.Exec(ctx, query, args...)
}
