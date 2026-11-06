package telemetry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

type TripObserver struct {
	logger    *slog.Logger
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewTripObserver(meter metric.Meter, logger *slog.Logger) (*TripObserver, error) {
	created, err := meter.Int64Counter("tripgo.trips.created", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create trips created metric: %w", err)
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, fmt.Errorf("create trips completed metric: %w", err)
	}

	return &TripObserver{
		logger:    logger,
		created:   created,
		completed: completed,
	}, nil
}

func (o *TripObserver) TripCreated(ctx context.Context, tripEntity *domainTrip.Trip) {
	o.created.Add(ctx, 1)
	o.logger.InfoContext(ctx, "trip created", "trip_id", uuid.UUID(tripEntity.ID()).String())
}

func (o *TripObserver) TripCompleted(ctx context.Context, tripEntity *domainTrip.Trip) {
	o.completed.Add(ctx, 1)
	o.logger.InfoContext(ctx, "trip completed", "trip_id", uuid.UUID(tripEntity.ID()).String())
}
