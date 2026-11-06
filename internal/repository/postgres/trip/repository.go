package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/a-pelikh/tripgo/internal/model"
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

func (r *repository) CreateTrip(ctx context.Context, tripID model.TripID, userID model.UserID, driverID model.DriverID, startPoint model.Coordinates, endPoint model.Coordinates, price int64, startedAt time.Time) (model.Trip, error) {
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
			tripID,
			userID,
			driverID,
			startPoint.Latitude(),
			startPoint.Longitude(),
			endPoint.Latitude(),
			endPoint.Longitude(),
			price,
			model.StatusActive,
			startedAt,
		).
		Suffix("RETURNING status, started_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build create trip query: %w", err)
	}

	var (
		persistedStatus    model.Status
		persistedStartedAt time.Time
	)

	if err = r.queryRow(ctx, query, args...).Scan(&persistedStatus, &persistedStartedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return model.Trip{}, model.ErrDriverBusy
		}

		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	return model.Trip{
		ID:         tripID,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      price,
		Status:     persistedStatus,
		StartedAt:  persistedStartedAt,
		FinishedAt: nil,
	}, nil
}

func (r *repository) RecordStatusChange(ctx context.Context, tripID model.TripID, fromStatus *model.Status, toStatus model.Status, reason string) error {
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

func (r *repository) GetTrip(ctx context.Context, tripID model.TripID) (model.Trip, error) {
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
		return model.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	trip, err := scanTrip(r.queryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, model.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (r *repository) FinishTrip(ctx context.Context, tripID model.TripID, finishedAt time.Time) (model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set("status", model.StatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{"id": uuid.UUID(tripID), "status": model.StatusActive}).
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
		return model.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	trip, err := scanTrip(r.queryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		existingTrip, getErr := r.GetTrip(ctx, tripID)
		if errors.Is(getErr, model.ErrTripNotFound) {
			return model.Trip{}, model.ErrTripNotFound
		}
		if getErr != nil {
			return model.Trip{}, fmt.Errorf("check trip after unsuccessful finish: %w", getErr)
		}
		if existingTrip.Status == model.StatusCompleted {
			return model.Trip{}, model.ErrTripCompleted
		}

		return model.Trip{}, fmt.Errorf("finish trip: unexpected trip status %q", existingTrip.Status)
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	return trip, nil
}

func scanTrip(row pgx.Row) (model.Trip, error) {
	var (
		id             uuid.UUID
		userID         uuid.UUID
		driverID       uuid.UUID
		startLatitude  float64
		startLongitude float64
		endLatitude    float64
		endLongitude   float64
		trip           model.Trip
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
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&finishedAt,
	); err != nil {
		return model.Trip{}, err
	}

	startPoint, err := model.NewCoordinates(startLatitude, startLongitude)
	if err != nil {
		return model.Trip{}, fmt.Errorf("restore trip start point: %w", err)
	}
	endPoint, err := model.NewCoordinates(endLatitude, endLongitude)
	if err != nil {
		return model.Trip{}, fmt.Errorf("restore trip end point: %w", err)
	}

	trip.ID = model.TripID(id)
	trip.UserID = model.UserID(userID)
	trip.DriverID = model.DriverID(driverID)
	trip.StartPoint = startPoint
	trip.EndPoint = endPoint
	if finishedAt.Valid {
		trip.FinishedAt = &finishedAt.Time
	}

	return trip, nil
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
