package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) Fetch(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Select(
		"id",
		"user_id",
		"driver_id",
		"start_latitude",
		"start_longitude",
		"end_latitude",
		"end_longitude",
		"price",
		"status",
		"started_at",
		"finished_at",
	).
		From("trips").
		Where(sq.Eq{"id": uuid.UUID(tripID)}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("build fetch trip query: %w", err)
	}

	tripEntity, err := scanTrip(r.queryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainTrip.Trip{}, domainTrip.ErrTripNotFound
	}
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("fetch trip: %w", err)
	}
	return tripEntity, nil
}
