package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/tx-manager"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func New(pool *pgxpool.Pool, queryTimeout time.Duration) *repository {
	return &repository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *repository) CreateTrip(ctx context.Context, tripEntity domainTrip.Trip) (domainTrip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trips").
		Columns(
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
		).
		Values(
			tripEntity.ID(),
			tripEntity.UserID(),
			tripEntity.DriverID(),
			tripEntity.StartPoint().Latitude(),
			tripEntity.StartPoint().Longitude(),
			tripEntity.EndPoint().Latitude(),
			tripEntity.EndPoint().Longitude(),
			tripEntity.Price(),
			tripEntity.Status(),
			tripEntity.StartedAt(),
		).
		Suffix("RETURNING status, started_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("build create trip query: %w", err)
	}

	var (
		persistedStatus    domainTrip.Status
		persistedStartedAt time.Time
	)

	if err = r.queryRow(ctx, query, args...).Scan(&persistedStatus, &persistedStartedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return domainTrip.Trip{}, domainTrip.ErrDriverBusy
		}

		return domainTrip.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	persistedTrip, err := domainTrip.New(
		tripEntity.ID(),
		tripEntity.UserID(),
		tripEntity.DriverID(),
		tripEntity.StartPoint(),
		tripEntity.EndPoint(),
		tripEntity.Price(),
		persistedStatus,
		persistedStartedAt,
		nil,
	)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("restore created trip: %w", err)
	}

	return persistedTrip, nil
}

func (r *repository) RecordStatusChange(ctx context.Context, tripID domainTrip.ID, fromStatus *domainTrip.Status, toStatus domainTrip.Status, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, fromStatus, toStatus, reason).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build record status change query: %w", err)
	}

	if _, err = r.exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record status change: %w", err)
	}

	return nil
}

func (r *repository) GetTrip(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error) {
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
		return domainTrip.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	tripEntity, err := scanTrip(r.queryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domainTrip.Trip{}, domainTrip.ErrTripNotFound
	}
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return tripEntity, nil
}

func (r *repository) FinishTrip(ctx context.Context, tripEntity domainTrip.Trip) (domainTrip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set("status", tripEntity.Status()).
		Set("finished_at", tripEntity.FinishedAt()).
		Set("updated_at", tripEntity.FinishedAt()).
		Where(sq.Eq{"id": uuid.UUID(tripEntity.ID()), "status": domainTrip.StatusActive}).
		Suffix(`RETURNING
			id,
			user_id,
			driver_id,
			start_latitude,
			start_longitude,
			end_latitude,
			end_longitude,
			price,
			status,
			started_at,
			finished_at`).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	persistedTrip, err := scanTrip(r.queryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		existingTrip, getErr := r.GetTrip(ctx, tripEntity.ID())
		if errors.Is(getErr, domainTrip.ErrTripNotFound) {
			return domainTrip.Trip{}, domainTrip.ErrTripNotFound
		}
		if getErr != nil {
			return domainTrip.Trip{}, fmt.Errorf("check trip after unsuccessful finish: %w", getErr)
		}
		if existingTrip.Status() == domainTrip.StatusCompleted {
			return domainTrip.Trip{}, domainTrip.ErrTripCompleted
		}

		return domainTrip.Trip{}, fmt.Errorf("finish trip: unexpected trip status %q", existingTrip.Status())
	}
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	return persistedTrip, nil
}

func scanTrip(row pgx.Row) (domainTrip.Trip, error) {
	var (
		id             uuid.UUID
		userID         uuid.UUID
		driverID       uuid.UUID
		startLatitude  float64
		startLongitude float64
		endLatitude    float64
		endLongitude   float64
		price          int64
		status         domainTrip.Status
		startedAt      time.Time
		finishedAt     pgtype.Timestamptz
	)

	if err := row.Scan(
		&id,
		&userID,
		&driverID,
		&startLatitude,
		&startLongitude,
		&endLatitude,
		&endLongitude,
		&price,
		&status,
		&startedAt,
		&finishedAt,
	); err != nil {
		return domainTrip.Trip{}, err
	}

	startPoint, err := domainTrip.NewCoordinates(startLatitude, startLongitude)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("restore trip start point: %w", err)
	}
	endPoint, err := domainTrip.NewCoordinates(endLatitude, endLongitude)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("restore trip end point: %w", err)
	}

	var domainFinishedAt *time.Time
	if finishedAt.Valid {
		domainFinishedAt = &finishedAt.Time
	}

	tripEntity, err := domainTrip.New(
		domainTrip.ID(id),
		domainTrip.UserID(userID),
		domainTrip.DriverID(driverID),
		startPoint,
		endPoint,
		price,
		status,
		startedAt,
		domainFinishedAt,
	)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("restore trip: %w", err)
	}

	return tripEntity, nil
}

func (r *repository) queryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if tx, ok := ctx.Value(tx_manager.TxKey{}).(pgx.Tx); ok {
		return tx.QueryRow(ctx, query, args...)
	}

	return r.pool.QueryRow(ctx, query, args...)
}

func (r *repository) exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := ctx.Value(tx_manager.TxKey{}).(pgx.Tx); ok {
		return tx.Exec(ctx, query, args...)
	}

	return r.pool.Exec(ctx, query, args...)
}

func (r *repository) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	return r.pool.Ping(ctx)
}
