package get_test

import (
	"context"
	"errors"
	"testing"
	"time"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	tripGet "github.com/a-pelikh/tripgo/internal/usecase/trip/get"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestUseCaseExecute(t *testing.T) {
	t.Parallel()

	tripEntity := newTrip(t)
	tests := []struct {
		name    string
		result  domainTrip.Trip
		err     error
		wantErr error
	}{
		{name: "success", result: tripEntity},
		{name: "not found", err: domainTrip.ErrTripNotFound, wantErr: domainTrip.ErrTripNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := NewMocktripRepository(ctrl)
			repo.EXPECT().Fetch(gomock.Any(), tripEntity.ID()).Return(tt.result, tt.err)

			result, err := tripGet.New(repo).Execute(context.Background(), tripEntity.ID())

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && result.ID() != tripEntity.ID() {
				t.Fatalf("trip id = %v, want %v", result.ID(), tripEntity.ID())
			}
		})
	}
}

func newTrip(t *testing.T) domainTrip.Trip {
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
