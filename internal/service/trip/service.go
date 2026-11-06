package trip

import (
	"context"
	"fmt"
	"time"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/tx-manager"
	"github.com/google/uuid"
)

const (
	tripCreatedReason   = "trip created"
	tripCompletedReason = "trip completed"
)

type tripRepository interface {
	CreateTrip(ctx context.Context, trip domainTrip.Trip) (domainTrip.Trip, error)
	GetTrip(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error)
	FinishTrip(ctx context.Context, trip domainTrip.Trip) (domainTrip.Trip, error)
	RecordStatusChange(ctx context.Context, tripID domainTrip.ID, fromStatus *domainTrip.Status, toStatus domainTrip.Status, reason string) error
	Ping(ctx context.Context) error
}

type service struct {
	txManager tx_manager.TxManager
	tripRepo  tripRepository
}

func New(txManager tx_manager.TxManager, tripRepo tripRepository) *service {
	return &service{
		txManager: txManager,
		tripRepo:  tripRepo,
	}
}

func (s *service) CreateTrip(ctx context.Context, userID domainTrip.UserID, driverID domainTrip.DriverID, startPoint domainTrip.Coordinates, endPoint domainTrip.Coordinates, price int64) (domainTrip.Trip, error) {
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

	if err := s.txManager.Do(ctx, func(ctx context.Context) error {
		if tripEntity, err = s.tripRepo.CreateTrip(ctx, tripEntity); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}
		if err = s.tripRepo.RecordStatusChange(ctx, tripEntity.ID(), nil, domainTrip.StatusActive, tripCreatedReason); err != nil {
			return fmt.Errorf("record initial trip status: %w", err)
		}

		return nil
	}); err != nil {
		return domainTrip.Trip{}, err
	}

	return tripEntity, nil
}

func (s *service) GetTrip(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error) {
	tripEntity, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return domainTrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return tripEntity, nil
}

func (s *service) FinishTrip(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error) {
	finishedAt := time.Now().UTC()
	fromStatus := domainTrip.StatusActive
	var tripEntity domainTrip.Trip

	if err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		if tripEntity, err = s.tripRepo.GetTrip(ctx, tripID); err != nil {
			return fmt.Errorf("get trip for finish: %w", err)
		}
		if err = tripEntity.Finish(finishedAt); err != nil {
			return fmt.Errorf("finish trip entity: %w", err)
		}
		if tripEntity, err = s.tripRepo.FinishTrip(ctx, tripEntity); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}
		if err = s.tripRepo.RecordStatusChange(ctx, tripID, &fromStatus, domainTrip.StatusCompleted, tripCompletedReason); err != nil {
			return fmt.Errorf("record completed trip status: %w", err)
		}

		return nil
	}); err != nil {
		return domainTrip.Trip{}, err
	}

	return tripEntity, nil
}

func (s *service) Ready(ctx context.Context) error {
	if err := s.tripRepo.Ping(ctx); err != nil {
		return fmt.Errorf("ping trip repository: %w", err)
	}
	return nil
}
