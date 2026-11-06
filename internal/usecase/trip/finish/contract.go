package finish

import (
	"context"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type tripRepository interface {
	Fetch(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error)
	Save(ctx context.Context, trip domainTrip.Trip) error
	SaveHistory(ctx context.Context, tripID domainTrip.ID, fromStatus *domainTrip.Status, toStatus domainTrip.Status, reason string) error
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
