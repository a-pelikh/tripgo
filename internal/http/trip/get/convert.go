package get

import (
	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/google/uuid"
)

func toAPI(tripEntity *domainTrip.Trip) api.Trip {
	return api.Trip{
		Id:       uuid.UUID(tripEntity.ID()),
		UserId:   uuid.UUID(tripEntity.UserID()),
		DriverId: uuid.UUID(tripEntity.DriverID()),
		StartPoint: api.Coordinates{
			Latitude:  tripEntity.StartPoint().Latitude(),
			Longitude: tripEntity.StartPoint().Longitude(),
		},
		EndPoint: api.Coordinates{
			Latitude:  tripEntity.EndPoint().Latitude(),
			Longitude: tripEntity.EndPoint().Longitude(),
		},
		Price:      tripEntity.Price(),
		Status:     api.TripStatus(tripEntity.Status()),
		StartedAt:  tripEntity.StartedAt(),
		FinishedAt: tripEntity.FinishedAt(),
	}
}
