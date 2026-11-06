package trip

import (
	"fmt"
	"time"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func scanTrip(row pgx.Row) (domainTrip.Trip, error) {
	var (
		id, userID, driverID          uuid.UUID
		startLatitude, startLongitude float64
		endLatitude, endLongitude     float64
		price                         int64
		status                        domainTrip.Status
		startedAt                     time.Time
		finishedAt                    pgtype.Timestamptz
	)
	if err := row.Scan(
		&id, &userID, &driverID,
		&startLatitude, &startLongitude,
		&endLatitude, &endLongitude,
		&price, &status, &startedAt, &finishedAt,
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
		domainTrip.ID(id), domainTrip.UserID(userID), domainTrip.DriverID(driverID),
		startPoint, endPoint, price, status, startedAt, domainFinishedAt,
	)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("restore trip: %w", err)
	}
	return tripEntity, nil
}
