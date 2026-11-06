//go:build integration

package trip_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/infra/postgres/transaction"
	tripRepository "github.com/a-pelikh/tripgo/internal/repository/trip"
)

const queryTimeout = 3 * time.Second

func TestRepositorySaveAndFetch(t *testing.T) {
	pool, repository := openRepository(t)
	tripEntity := newTrip(t)
	cleanupTrip(t, pool, tripEntity.ID())

	ctx := context.Background()
	if err := repository.Save(ctx, tripEntity); err != nil {
		t.Fatalf("Save() active trip error = %v", err)
	}
	if err := repository.SaveHistory(ctx, tripEntity.ID(), nil, domainTrip.StatusActive, ""); err != nil {
		t.Fatalf("SaveHistory() active trip error = %v", err)
	}

	stored, err := repository.Fetch(ctx, tripEntity.ID())
	if err != nil {
		t.Fatalf("Fetch() active trip error = %v", err)
	}
	assertTrip(t, stored, tripEntity)

	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	if err := tripEntity.Finish(finishedAt); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if err := repository.Save(ctx, tripEntity); err != nil {
		t.Fatalf("Save() completed trip error = %v", err)
	}
	fromStatus := domainTrip.StatusActive
	if err := repository.SaveHistory(ctx, tripEntity.ID(), &fromStatus, domainTrip.StatusCompleted, ""); err != nil {
		t.Fatalf("SaveHistory() completed trip error = %v", err)
	}

	stored, err = repository.Fetch(ctx, tripEntity.ID())
	if err != nil {
		t.Fatalf("Fetch() completed trip error = %v", err)
	}
	assertTrip(t, stored, tripEntity)
}

func TestTransactionRollsBackTripAndHistory(t *testing.T) {
	pool, repository := openRepository(t)
	tripEntity := newTrip(t)
	cleanupTrip(t, pool, tripEntity.ID())

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	txManager := transaction.New(logger, pool, queryTimeout)
	err := txManager.Do(context.Background(), func(ctx context.Context) error {
		if err := repository.Save(ctx, tripEntity); err != nil {
			return err
		}

		return repository.SaveHistory(ctx, tripEntity.ID(), nil, domainTrip.Status("invalid"), "")
	})
	if err == nil {
		t.Fatal("Do() error = nil, want history constraint error")
	}

	_, err = repository.Fetch(context.Background(), tripEntity.ID())
	if !errors.Is(err, domainTrip.ErrTripNotFound) {
		t.Fatalf("Fetch() after rollback error = %v, want ErrTripNotFound", err)
	}
}

func openRepository(t *testing.T) (*pgxpool.Pool, *tripRepository.Repository) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	return pool, tripRepository.New(pool, queryTimeout)
}

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, tripID domainTrip.ID) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		if _, err := pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", uuid.UUID(tripID)); err != nil {
			t.Errorf("clean up trip: %v", err)
		}
	})
}

func newTrip(t *testing.T) domainTrip.Trip {
	t.Helper()

	startPoint, err := domainTrip.NewCoordinates(59.9398, 30.3146)
	if err != nil {
		t.Fatalf("NewCoordinates() start error = %v", err)
	}
	endPoint, err := domainTrip.NewCoordinates(59.929, 30.3626)
	if err != nil {
		t.Fatalf("NewCoordinates() end error = %v", err)
	}
	tripEntity, err := domainTrip.New(
		domainTrip.ID(uuid.New()),
		domainTrip.UserID(uuid.New()),
		domainTrip.DriverID(uuid.New()),
		startPoint,
		endPoint,
		1450,
		domainTrip.StatusActive,
		time.Now().UTC().Truncate(time.Microsecond),
		nil,
	)
	if err != nil {
		t.Fatalf("trip.New() error = %v", err)
	}

	return tripEntity
}

func assertTrip(t *testing.T, got, want domainTrip.Trip) {
	t.Helper()

	if got.ID() != want.ID() || got.UserID() != want.UserID() || got.DriverID() != want.DriverID() {
		t.Fatalf("stored identity = (%v, %v, %v), want (%v, %v, %v)", got.ID(), got.UserID(), got.DriverID(), want.ID(), want.UserID(), want.DriverID())
	}
	if got.Status() != want.Status() || got.Price() != want.Price() || !got.StartedAt().Equal(want.StartedAt()) {
		t.Fatalf("stored trip = status %q, price %d, started %v; want status %q, price %d, started %v", got.Status(), got.Price(), got.StartedAt(), want.Status(), want.Price(), want.StartedAt())
	}
	if got.StartPoint() != want.StartPoint() || got.EndPoint() != want.EndPoint() {
		t.Fatalf("stored route = (%v, %v), want (%v, %v)", got.StartPoint(), got.EndPoint(), want.StartPoint(), want.EndPoint())
	}
	if (got.FinishedAt() == nil) != (want.FinishedAt() == nil) {
		t.Fatalf("stored finished_at = %v, want %v", got.FinishedAt(), want.FinishedAt())
	}
	if got.FinishedAt() != nil && !got.FinishedAt().Equal(*want.FinishedAt()) {
		t.Fatalf("stored finished_at = %v, want %v", got.FinishedAt(), want.FinishedAt())
	}
}
