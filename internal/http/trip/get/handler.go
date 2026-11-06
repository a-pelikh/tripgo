package get

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/http/problem"
	"github.com/google/uuid"
)

type Handler struct {
	logger  *slog.Logger
	useCase useCase
}

func New(logger *slog.Logger, useCase useCase) *Handler {
	return &Handler{
		logger:  logger.With("component", "get_trip_handler"),
		useCase: useCase,
	}
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if tripID == uuid.Nil {
		problem.Write(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	tripEntity, err := h.useCase.Execute(r.Context(), domainTrip.ID(tripID))
	if err != nil {
		if errors.Is(err, domainTrip.ErrTripNotFound) {
			problem.Write(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
			return
		}

		h.logger.ErrorContext(r.Context(), "failed to get trip", "error", err)
		problem.Write(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toAPI(&tripEntity))
}
