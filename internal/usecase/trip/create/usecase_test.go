package create_test

import (
	"context"
	"errors"
	"testing"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	tripCreate "github.com/a-pelikh/tripgo/internal/usecase/trip/create"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestUseCaseExecuteSuccess(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	expectTransaction(tx)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().SaveHistory(gomock.Any(), gomock.Any(), nil, domainTrip.StatusActive, "trip created").Return(nil)
	start, end := coordinates(t)

	result, err := tripCreate.New(tx, repo).Execute(
		context.Background(),
		domainTrip.UserID(uuid.New()),
		domainTrip.DriverID(uuid.New()),
		start,
		end,
		1450,
	)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status() != domainTrip.StatusActive {
		t.Fatalf("status = %q, want active", result.Status())
	}
}

func TestUseCaseExecuteDriverBusy(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)
	expectTransaction(tx)
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(domainTrip.ErrDriverBusy)
	start, end := coordinates(t)

	_, err := tripCreate.New(tx, repo).Execute(
		context.Background(),
		domainTrip.UserID(uuid.New()),
		domainTrip.DriverID(uuid.New()),
		start,
		end,
		1450,
	)

	if !errors.Is(err, domainTrip.ErrDriverBusy) {
		t.Fatalf("Execute() error = %v, want ErrDriverBusy", err)
	}
}

func expectTransaction(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	)
}

func coordinates(t *testing.T) (domainTrip.Coordinates, domainTrip.Coordinates) {
	t.Helper()
	start, err := domainTrip.NewCoordinates(59.9, 30.3)
	if err != nil {
		t.Fatalf("create start coordinates: %v", err)
	}
	end, err := domainTrip.NewCoordinates(59.8, 30.4)
	if err != nil {
		t.Fatalf("create end coordinates: %v", err)
	}
	return start, end
}
