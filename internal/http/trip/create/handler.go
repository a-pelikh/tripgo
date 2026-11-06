package create

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/http/domainerror"
)

type Handler struct {
	logger  *slog.Logger
	useCase useCase
}

func New(logger *slog.Logger, useCase useCase) *Handler {
	return &Handler{
		logger:  logger.With("component", "create_trip_handler"),
		useCase: useCase,
	}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var request api.CreateTripJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		domainerror.InvalidRequest(w, r, "Request body is invalid")
		return
	}

	if request.UserId == uuid.Nil || request.DriverId == uuid.Nil || request.Price < 0 {
		domainerror.InvalidRequest(w, r, "Request validation failed")
		return
	}

	startPoint, err := domainTrip.NewCoordinates(request.StartPoint.Latitude, request.StartPoint.Longitude)
	if err != nil {
		domainerror.InvalidRequest(w, r, "Start point is invalid")
		return
	}

	endPoint, err := domainTrip.NewCoordinates(request.EndPoint.Latitude, request.EndPoint.Longitude)
	if err != nil {
		domainerror.InvalidRequest(w, r, "End point is invalid")
		return
	}

	tripEntity, err := h.useCase.Execute(
		r.Context(),
		domainTrip.UserID(request.UserId),
		domainTrip.DriverID(request.DriverId),
		startPoint,
		endPoint,
		request.Price,
	)
	if err != nil {
		if domainerror.Write(w, r, err) >= http.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "failed to create trip", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+uuid.UUID(tripEntity.ID()).String())
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAPI(&tripEntity))
}
