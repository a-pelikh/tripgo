package ready

import (
	"encoding/json"
	"net/http"

	"github.com/a-pelikh/tripgo/api"
)

type Handler struct {
	checker readinessChecker
}

func New(checker readinessChecker) *Handler {
	return &Handler{checker: checker}
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := api.HealthResponse{Status: api.Ok}
	if err := h.checker.Ping(r.Context()); err != nil {
		response.Status = api.Unavailable
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(response)
}
