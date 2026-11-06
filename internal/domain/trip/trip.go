package trip

import (
	"time"

	"github.com/google/uuid"
)

type (
	ID       uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID
	Status   string
)

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Trip struct {
	id         ID
	userID     UserID
	driverID   DriverID
	startPoint Coordinates
	endPoint   Coordinates
	price      int64
	status     Status
	startedAt  time.Time
	finishedAt *time.Time
}

func New(
	id ID,
	userID UserID,
	driverID DriverID,
	startPoint Coordinates,
	endPoint Coordinates,
	price int64,
	status Status,
	startedAt time.Time,
	finishedAt *time.Time,
) (Trip, error) {
	if uuid.UUID(id) == uuid.Nil {
		return Trip{}, invalid("id", "must not be empty")
	}
	if uuid.UUID(userID) == uuid.Nil {
		return Trip{}, invalid("user_id", "must not be empty")
	}
	if uuid.UUID(driverID) == uuid.Nil {
		return Trip{}, invalid("driver_id", "must not be empty")
	}
	if price < 0 {
		return Trip{}, invalid("price", "must not be negative")
	}
	if startedAt.IsZero() {
		return Trip{}, invalid("started_at", "must not be empty")
	}

	switch status {
	case StatusActive:
		if finishedAt != nil {
			return Trip{}, invalid("finished_at", "must be empty for an active trip")
		}
	case StatusCompleted:
		if finishedAt == nil {
			return Trip{}, invalid("finished_at", "must be set for a completed trip")
		}
	default:
		return Trip{}, invalid("status", "must be active or completed")
	}

	return Trip{
		id:         id,
		userID:     userID,
		driverID:   driverID,
		startPoint: startPoint,
		endPoint:   endPoint,
		price:      price,
		status:     status,
		startedAt:  startedAt,
		finishedAt: cloneTime(finishedAt),
	}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.status != StatusActive {
		return ErrTripCompleted
	}
	if at.IsZero() {
		return invalid("finished_at", "must not be empty")
	}

	finishedAt := at.UTC()
	t.status = StatusCompleted
	t.finishedAt = &finishedAt

	return nil
}

func (t *Trip) ID() ID {
	return t.id
}

func (t *Trip) UserID() UserID {
	return t.userID
}

func (t *Trip) DriverID() DriverID {
	return t.driverID
}

func (t *Trip) StartPoint() Coordinates {
	return t.startPoint
}

func (t *Trip) EndPoint() Coordinates {
	return t.endPoint
}

func (t *Trip) Price() int64 {
	return t.price
}

func (t *Trip) Status() Status {
	return t.status
}

func (t *Trip) StartedAt() time.Time {
	return t.startedAt
}

func (t *Trip) FinishedAt() *time.Time {
	return cloneTime(t.finishedAt)
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
