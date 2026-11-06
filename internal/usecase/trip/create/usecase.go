package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

const tripCreatedReason = "trip created"

type UseCase struct {
	txManager txManager
	tripRepo  tripRepository
	observer  observer
}

func New(txManager txManager, tripRepo tripRepository, observer observer) *UseCase {
	return &UseCase{txManager: txManager, tripRepo: tripRepo, observer: observer}
}

func (uc *UseCase) Execute(
	ctx context.Context,
	userID domainTrip.UserID,
	driverID domainTrip.DriverID,
	startPoint domainTrip.Coordinates,
	endPoint domainTrip.Coordinates,
	price int64,
) (domainTrip.Trip, error) {
	tripEntity, err := domainTrip.New(
		domainTrip.ID(uuid.New()),
		userID,
		driverID,
		startPoint,
		endPoint,
		price,
		domainTrip.StatusActive,
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("create trip entity: %w", err)
	}

	if err := uc.txManager.Do(ctx, func(ctx context.Context) error {
		if err = uc.tripRepo.Save(ctx, tripEntity); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}
		if err = uc.tripRepo.SaveHistory(ctx, tripEntity.ID(), nil, domainTrip.StatusActive, tripCreatedReason); err != nil {
			return fmt.Errorf("record initial trip status: %w", err)
		}
		return nil
	}); err != nil {
		return domainTrip.Trip{}, err
	}

	uc.observer.TripCreated(ctx, &tripEntity)
	return tripEntity, nil
}
