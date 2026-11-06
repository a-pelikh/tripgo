package model

import (
	"time"

	"github.com/google/uuid"
)

type TripID uuid.UUID

type Trip struct {
	ID         TripID
	UserID     UserID
	DriverID   DriverID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}
