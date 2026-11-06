package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

func (r *Repository) Save(ctx context.Context, tripEntity domainTrip.Trip) error {
	switch tripEntity.Status() {
	case domainTrip.StatusActive:
		return r.insert(ctx, tripEntity)
	case domainTrip.StatusCompleted:
		return r.update(ctx, tripEntity)
	default:
		return fmt.Errorf("save trip: unsupported status %q", tripEntity.Status())
	}
}

func (r *Repository) SaveHistory(ctx context.Context, tripID domainTrip.ID, fromStatus *domainTrip.Status, toStatus domainTrip.Status, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, fromStatus, toStatus, reason).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build save trip history query: %w", err)
	}
	if _, err = r.exec(ctx, query, args...); err != nil {
		return fmt.Errorf("save trip history: %w", err)
	}
	return nil
}

func (r *Repository) insert(ctx context.Context, tripEntity domainTrip.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
		).
		Values(
			tripEntity.ID(), tripEntity.UserID(), tripEntity.DriverID(),
			tripEntity.StartPoint().Latitude(), tripEntity.StartPoint().Longitude(),
			tripEntity.EndPoint().Latitude(), tripEntity.EndPoint().Longitude(),
			tripEntity.Price(), tripEntity.Status(), tripEntity.StartedAt(),
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip query: %w", err)
	}
	if _, err = r.exec(ctx, query, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return domainTrip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *Repository) update(ctx context.Context, tripEntity domainTrip.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set("status", tripEntity.Status()).
		Set("finished_at", tripEntity.FinishedAt()).
		Set("updated_at", tripEntity.FinishedAt()).
		Where(sq.Eq{"id": uuid.UUID(tripEntity.ID()), "status": domainTrip.StatusActive}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip query: %w", err)
	}

	tag, err := r.exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}

	existingTrip, err := r.Fetch(ctx, tripEntity.ID())
	if err != nil {
		return fmt.Errorf("fetch trip after unsuccessful update: %w", err)
	}
	if existingTrip.Status() == domainTrip.StatusCompleted {
		return domainTrip.ErrTripCompleted
	}
	return fmt.Errorf("update trip: unexpected status %q", existingTrip.Status())
}
