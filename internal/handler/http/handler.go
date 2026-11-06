package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/google/uuid"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type tripService interface {
	CreateTrip(ctx context.Context, userID domainTrip.UserID, driverID domainTrip.DriverID, startPoint domainTrip.Coordinates, endPoint domainTrip.Coordinates, price int64) (domainTrip.Trip, error)
	GetTrip(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error)
	FinishTrip(ctx context.Context, tripID domainTrip.ID) (domainTrip.Trip, error)
	Ready(ctx context.Context) error
}

type handler struct {
	logger *slog.Logger
	trip   tripService
}

func New(logger *slog.Logger, tripServ tripService) api.ServerInterface {
	return &handler{
		logger: logger.With("component", "http_handler"),
		trip:   tripServ,
	}
}

func NewRouter(logger *slog.Logger, tripServ tripService) (http.Handler, error) {
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
		New(logger, tripServ),
		api.ChiServerOptions{
			Middlewares:      []api.MiddlewareFunc{validator},
			ErrorHandlerFunc: HandleBindingError,
		},
	), nil
}

func (h *handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var request api.CreateTripJSONRequestBody
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request body is invalid")
		return
	}

	if request.UserId == uuid.Nil || request.DriverId == uuid.Nil || request.Price < 0 {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	startPoint, err := domainTrip.NewCoordinates(request.StartPoint.Latitude, request.StartPoint.Longitude)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Start point is invalid")
		return
	}

	endPoint, err := domainTrip.NewCoordinates(request.EndPoint.Latitude, request.EndPoint.Longitude)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "End point is invalid")
		return
	}

	tripEntity, err := h.trip.CreateTrip(r.Context(), domainTrip.UserID(request.UserId), domainTrip.DriverID(request.DriverId), startPoint, endPoint, request.Price)
	if err != nil {
		if errors.Is(err, domainTrip.ErrDriverBusy) {
			writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
			return
		}

		h.logger.ErrorContext(r.Context(), "failed to create trip", "error", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+uuid.UUID(tripEntity.ID()).String())
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mapTripToAPI(tripEntity))

}

func (h *handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if tripID == uuid.Nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	tripEntity, err := h.trip.GetTrip(r.Context(), domainTrip.ID(tripID))
	if err != nil {
		if errors.Is(err, domainTrip.ErrTripNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
			return
		}

		h.logger.ErrorContext(r.Context(), "failed to get trip", "error", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToAPI(tripEntity))
}

func (h *handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if tripID == uuid.Nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	tripEntity, err := h.trip.FinishTrip(r.Context(), domainTrip.ID(tripID))
	if err != nil {
		switch {
		case errors.Is(err, domainTrip.ErrTripNotFound):
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
		case errors.Is(err, domainTrip.ErrTripCompleted):
			writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip")
		default:
			h.logger.ErrorContext(r.Context(), "failed to finish trip", "error", err)
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToAPI(tripEntity))
}

func (h *handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.Ok,
	})
}

func (h *handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := api.HealthResponse{
		Status: api.Ok,
	}
	if err := h.trip.Ready(r.Context()); err != nil {
		response.Status = api.Unavailable
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(response)
}

func HandleBindingError(w http.ResponseWriter, r *http.Request, _ error) {
	writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
}

func mapTripToAPI(tripEntity domainTrip.Trip) api.Trip {
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

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code string, title string, detail string) {
	instance := r.URL.Path

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
}
