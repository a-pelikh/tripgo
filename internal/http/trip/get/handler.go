package get

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
		logger:  logger.With("component", "get_trip_handler"),
		useCase: useCase,
	}
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if tripID == uuid.Nil {
		domainerror.InvalidRequest(w, r, "Request validation failed")
		return
	}

	tripEntity, err := h.useCase.Execute(r.Context(), domainTrip.ID(tripID))
	if err != nil {
		if domainerror.Write(w, r, err) >= http.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "failed to get trip", "error", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toAPI(&tripEntity))
}
