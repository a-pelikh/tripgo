package create

import (
	"context"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type useCase interface {
	Execute(
		ctx context.Context,
		userID domainTrip.UserID,
		driverID domainTrip.DriverID,
		startPoint domainTrip.Coordinates,
		endPoint domainTrip.Coordinates,
		price int64,
	) (domainTrip.Trip, error)
}
