package get

import (
	"context"
	"fmt"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

type UseCase struct {
	tripRepo tripRepository
}

func New(tripRepo tripRepository) *UseCase {
	return &UseCase{tripRepo: tripRepo}
}

func (uc *UseCase) Execute(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error) {
	tripEntity, err := uc.tripRepo.Fetch(ctx, tripID)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return tripEntity, nil
}
