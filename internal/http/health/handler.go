package health

import (
	"encoding/json"
	"net/http"

	"github.com/a-pelikh/tripgo/api"
)

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}
