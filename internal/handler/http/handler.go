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
	"github.com/a-pelikh/tripgo/internal/model"
	"github.com/google/uuid"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type tripService interface {
	CreateTrip(ctx context.Context, userID model.UserID, driverID model.DriverID, startPoint model.Coordinates, endPoint model.Coordinates, price int64) (model.Trip, error)
	GetTrip(ctx context.Context, tripID model.TripID) (model.Trip, error)
	FinishTrip(ctx context.Context, tripID model.TripID) (model.Trip, error)
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

	startPoint, err := model.NewCoordinates(request.StartPoint.Latitude, request.StartPoint.Longitude)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Start point is invalid")
		return
	}

	endPoint, err := model.NewCoordinates(request.EndPoint.Latitude, request.EndPoint.Longitude)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "End point is invalid")
		return
	}

	trip, err := h.trip.CreateTrip(r.Context(), model.UserID(request.UserId), model.DriverID(request.DriverId), startPoint, endPoint, request.Price)
	if err != nil {
		if errors.Is(err, model.ErrDriverBusy) {
			writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
			return
		}

		h.logger.ErrorContext(r.Context(), "failed to create trip", "error", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+uuid.UUID(trip.ID).String())
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mapTripToAPI(trip))

}

func (h *handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if tripID == uuid.Nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	trip, err := h.trip.GetTrip(r.Context(), model.TripID(tripID))
	if err != nil {
		if errors.Is(err, model.ErrTripNotFound) {
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
			return
		}

		h.logger.ErrorContext(r.Context(), "failed to get trip", "error", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToAPI(trip))
}

func (h *handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if tripID == uuid.Nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	trip, err := h.trip.FinishTrip(r.Context(), model.TripID(tripID))
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTripNotFound):
			writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
		case errors.Is(err, model.ErrTripCompleted):
			writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip")
		default:
			h.logger.ErrorContext(r.Context(), "failed to finish trip", "error", err)
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToAPI(trip))
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

func mapTripToAPI(trip model.Trip) api.Trip {
	return api.Trip{
		Id:       uuid.UUID(trip.ID),
		UserId:   uuid.UUID(trip.UserID),
		DriverId: uuid.UUID(trip.DriverID),
		StartPoint: api.Coordinates{
			Latitude:  trip.StartPoint.Latitude(),
			Longitude: trip.StartPoint.Longitude(),
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndPoint.Latitude(),
			Longitude: trip.EndPoint.Longitude(),
		},
		Price:      trip.Price,
		Status:     api.TripStatus(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
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
