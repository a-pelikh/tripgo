package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/http/health"
	"github.com/a-pelikh/tripgo/internal/http/problem"
	"github.com/a-pelikh/tripgo/internal/http/ready"
	tripCreate "github.com/a-pelikh/tripgo/internal/http/trip/create"
	tripFinish "github.com/a-pelikh/tripgo/internal/http/trip/finish"
	tripGet "github.com/a-pelikh/tripgo/internal/http/trip/get"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type createUseCase interface {
	Execute(ctx context.Context, userID domainTrip.UserID, driverID domainTrip.DriverID, startPoint domainTrip.Coordinates, endPoint domainTrip.Coordinates, price int64) (domainTrip.Trip, error)
}

type tripUseCase interface {
	Execute(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error)
}

type readinessChecker interface {
	Ping(ctx context.Context) error
}

type handler struct {
	createHandler *tripCreate.Handler
	getHandler    *tripGet.Handler
	finishHandler *tripFinish.Handler
	healthHandler *health.Handler
	readyHandler  *ready.Handler
}

func New(logger *slog.Logger, createUC createUseCase, getUC, finishUC tripUseCase, checker readinessChecker) api.ServerInterface {
	return &handler{
		createHandler: tripCreate.New(logger, createUC),
		getHandler:    tripGet.New(logger, getUC),
		finishHandler: tripFinish.New(logger, finishUC),
		healthHandler: health.New(),
		readyHandler:  ready.New(checker),
	}
}

func NewRouter(logger *slog.Logger, createUC createUseCase, getUC, finishUC tripUseCase, checker readinessChecker) (http.Handler, error) {
	spec, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}

	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(
		spec,
		&nethttpmiddleware.Options{
			DoNotValidateServers: true,
			ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ nethttpmiddleware.ErrorHandlerOpts) {
				HandleBindingError(w, r, err)
			},
		},
	)

	return api.HandlerWithOptions(
		New(logger, createUC, getUC, finishUC, checker),
		api.ChiServerOptions{
			Middlewares:      []api.MiddlewareFunc{validator},
			ErrorHandlerFunc: HandleBindingError,
		},
	), nil
}

func (h *handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	h.createHandler.CreateTrip(w, r, params)
}

func (h *handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	h.getHandler.GetTrip(w, r, tripID)
}

func (h *handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	h.finishHandler.FinishTrip(w, r, tripID)
}

func (h *handler) Health(w http.ResponseWriter, r *http.Request) {
	h.healthHandler.Health(w, r)
}

func (h *handler) Ready(w http.ResponseWriter, r *http.Request) {
	h.readyHandler.Ready(w, r)
}

func HandleBindingError(w http.ResponseWriter, r *http.Request, _ error) {
	problem.Write(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
}
