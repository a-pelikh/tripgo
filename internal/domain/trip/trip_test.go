package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/a-pelikh/tripgo/internal/domain/trip"
)

func TestTripFinish(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.November, 6, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)
	entity := newTrip(t, startedAt)

	if err := entity.Finish(finishedAt); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if entity.Status() != trip.StatusCompleted {
		t.Fatalf("Status() = %q, want %q", entity.Status(), trip.StatusCompleted)
	}
	if entity.FinishedAt() == nil || !entity.FinishedAt().Equal(finishedAt) {
		t.Fatalf("FinishedAt() = %v, want %v", entity.FinishedAt(), finishedAt)
	}
	if err := entity.Finish(finishedAt.Add(time.Hour)); !errors.Is(err, trip.ErrTripCompleted) {
		t.Fatalf("second Finish() error = %v, want ErrTripCompleted", err)
	}
}

func TestTripFinishRejectsZeroTime(t *testing.T) {
	t.Parallel()

	entity := newTrip(t, time.Now().UTC())

	var validationErr trip.ValidationError
	if err := entity.Finish(time.Time{}); !errors.As(err, &validationErr) {
		t.Fatalf("Finish() error = %v, want ValidationError", err)
	}
	if entity.Status() != trip.StatusActive || entity.FinishedAt() != nil {
		t.Fatal("Finish() changed the trip after validation failure")
	}
}

func TestNewValidatesStateInvariant(t *testing.T) {
	t.Parallel()

	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(time.Hour)
	startPoint := newCoordinates(t, 59.9398, 30.3146)
	endPoint := newCoordinates(t, 59.9290, 30.3626)

	tests := []struct {
		name       string
		price      int64
		status     trip.Status
		finishedAt *time.Time
		field      string
	}{
		{
			name:       "active trip has finish time",
			price:      1450,
			status:     trip.StatusActive,
			finishedAt: &finishedAt,
			field:      "finished_at",
		},
		{
			name:   "completed trip has no finish time",
			price:  1450,
			status: trip.StatusCompleted,
			field:  "finished_at",
		},
		{
			name:   "negative price",
			price:  -1,
			status: trip.StatusActive,
			field:  "price",
		},
		{
			name:   "unknown status",
			price:  1450,
			status: "cancelled",
			field:  "status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := trip.New(
				trip.ID(uuid.New()),
				trip.UserID(uuid.New()),
				trip.DriverID(uuid.New()),
				startPoint,
				endPoint,
				tt.price,
				tt.status,
				startedAt,
				tt.finishedAt,
			)

			var validationErr trip.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("New() error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("ValidationError.Field = %q, want %q", validationErr.Field, tt.field)
			}
		})
	}
}

func newTrip(t *testing.T, startedAt time.Time) trip.Trip {
	t.Helper()

	startPoint := newCoordinates(t, 59.9398, 30.3146)
	endPoint := newCoordinates(t, 59.9290, 30.3626)
	entity, err := trip.New(
		trip.ID(uuid.New()),
		trip.UserID(uuid.New()),
		trip.DriverID(uuid.New()),
		startPoint,
		endPoint,
		1450,
		trip.StatusActive,
		startedAt,
		nil,
	)
	if err != nil {
		t.Fatalf("trip.New() error = %v", err)
	}

	return entity
}

func newCoordinates(t *testing.T, latitude, longitude float64) trip.Coordinates {
	t.Helper()

	coordinates, err := trip.NewCoordinates(latitude, longitude)
	if err != nil {
		t.Fatalf("NewCoordinates() error = %v", err)
	}

	return coordinates
}
