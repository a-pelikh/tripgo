package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	tripFinish "github.com/a-pelikh/tripgo/internal/usecase/trip/finish"
)

func TestUseCaseExecuteSuccess(t *testing.T) {
	t.Parallel()

	tripEntity := activeTrip(t)
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	observer := NewMockobserver(ctrl)
	expectTransaction(tx)
	repo.EXPECT().Fetch(gomock.Any(), tripEntity.ID()).Return(tripEntity, nil)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().SaveHistory(gomock.Any(), tripEntity.ID(), gomock.Any(), domainTrip.StatusCompleted, "trip completed").Return(nil)
	observer.EXPECT().TripCompleted(gomock.Any(), gomock.Any())

	result, err := tripFinish.New(tx, repo, observer).Execute(context.Background(), tripEntity.ID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status() != domainTrip.StatusCompleted || result.FinishedAt() == nil {
		t.Fatalf("finished trip = status %q, finished_at %v", result.Status(), result.FinishedAt())
	}
}

func TestUseCaseExecuteCompletedTrip(t *testing.T) {
	t.Parallel()

	tripEntity := activeTrip(t)
	if err := tripEntity.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("finish trip: %v", err)
	}
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	observer := NewMockobserver(ctrl)
	expectTransaction(tx)
	repo.EXPECT().Fetch(gomock.Any(), tripEntity.ID()).Return(tripEntity, nil)

	_, err := tripFinish.New(tx, repo, observer).Execute(context.Background(), tripEntity.ID())

	if !errors.Is(err, domainTrip.ErrTripCompleted) {
		t.Fatalf("Execute() error = %v, want ErrTripCompleted", err)
	}
}

func expectTransaction(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)
}

func activeTrip(t *testing.T) domainTrip.Trip {
	t.Helper()
	start, _ := domainTrip.NewCoordinates(59.9, 30.3)
	end, _ := domainTrip.NewCoordinates(59.8, 30.4)
	entity, err := domainTrip.New(
		domainTrip.ID(uuid.New()),
		domainTrip.UserID(uuid.New()),
		domainTrip.DriverID(uuid.New()),
		start,
		end,
		1450,
		domainTrip.StatusActive,
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}
	return entity
}
