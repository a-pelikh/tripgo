package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/a-pelikh/tripgo/internal/model"
	"github.com/a-pelikh/tripgo/internal/tx-manager"
	"github.com/google/uuid"
)

const (
	tripCreatedReason   = "trip created"
	tripCompletedReason = "trip completed"
)

type tripRepository interface {
	CreateTrip(ctx context.Context, tripID model.TripID, userID model.UserID, driverID model.DriverID, startPoint model.Coordinates, endPoint model.Coordinates, price int64, startedAt time.Time) (model.Trip, error)
	GetTrip(ctx context.Context, tripID model.TripID) (model.Trip, error)
	FinishTrip(ctx context.Context, tripID model.TripID, finishedAt time.Time) (model.Trip, error)
	RecordStatusChange(ctx context.Context, tripID model.TripID, fromStatus *model.Status, toStatus model.Status, reason string) error
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

func (s *service) CreateTrip(ctx context.Context, userID model.UserID, driverID model.DriverID, startPoint model.Coordinates, endPoint model.Coordinates, price int64) (model.Trip, error) {
	tripID := model.TripID(uuid.New())
	startedAt := time.Now().UTC()
	var trip model.Trip
	if err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		if trip, err = s.tripRepo.CreateTrip(ctx, tripID, userID, driverID, startPoint, endPoint, price, startedAt); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}
		if err = s.tripRepo.RecordStatusChange(ctx, tripID, nil, model.StatusActive, tripCreatedReason); err != nil {
			return fmt.Errorf("record initial trip status: %w", err)
		}

		return nil
	}); err != nil {
		return model.Trip{}, err
	}

	return trip, nil
}

func (s *service) GetTrip(ctx context.Context, tripID model.TripID) (model.Trip, error) {
	trip, err := s.tripRepo.GetTrip(ctx, tripID)
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (s *service) FinishTrip(ctx context.Context, tripID model.TripID) (model.Trip, error) {
	finishedAt := time.Now().UTC()
	fromStatus := model.StatusActive
	var trip model.Trip

	if err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		if trip, err = s.tripRepo.FinishTrip(ctx, tripID, finishedAt); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}
		if err = s.tripRepo.RecordStatusChange(ctx, tripID, &fromStatus, model.StatusCompleted, tripCompletedReason); err != nil {
			return fmt.Errorf("record completed trip status: %w", err)
		}

		return nil
	}); err != nil {
		return model.Trip{}, err
	}

	return trip, nil
}

func (s *service) Ready(ctx context.Context) error {
	if err := s.tripRepo.Ping(ctx); err != nil {
		return fmt.Errorf("ping trip repository: %w", err)
	}
	return nil
}
