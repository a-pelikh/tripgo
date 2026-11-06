package finish

import (
	"context"
	"fmt"
	"time"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

const tripCompletedReason = "trip completed"

type UseCase struct {
	txManager txManager
	tripRepo  tripRepository
	observer  observer
}

func New(txManager txManager, tripRepo tripRepository, observer observer) *UseCase {
	return &UseCase{txManager: txManager, tripRepo: tripRepo, observer: observer}
}

func (uc *UseCase) Execute(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error) {
	fromStatus := domainTrip.StatusActive
	var tripEntity domainTrip.Trip

	if err := uc.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		if tripEntity, err = uc.tripRepo.Fetch(ctx, tripID); err != nil {
			return fmt.Errorf("get trip for finish: %w", err)
		}
		if err = tripEntity.Finish(time.Now().UTC()); err != nil {
			return fmt.Errorf("finish trip entity: %w", err)
		}
		if err = uc.tripRepo.Save(ctx, tripEntity); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}
		if err = uc.tripRepo.SaveHistory(ctx, tripID, &fromStatus, domainTrip.StatusCompleted, tripCompletedReason); err != nil {
			return fmt.Errorf("record completed trip status: %w", err)
		}
		return nil
	}); err != nil {
		return domainTrip.Trip{}, err
	}

	uc.observer.TripCompleted(ctx, &tripEntity)
	return tripEntity, nil
}
