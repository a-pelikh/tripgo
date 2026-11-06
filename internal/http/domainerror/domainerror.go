package domainerror

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
)

func Write(w http.ResponseWriter, r *http.Request, err error) int {
	status, code, title, detail := mapError(err)
	write(w, r, status, code, title, detail)
	return status
}

func InvalidRequest(w http.ResponseWriter, r *http.Request, detail string) {
	write(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", detail)
}

func Internal(w http.ResponseWriter, r *http.Request) {
	write(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
}

func mapError(err error) (int, string, string, string) {
	switch {
	case errors.Is(err, domainTrip.ErrTripNotFound):
		return http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found"
	case errors.Is(err, domainTrip.ErrDriverBusy):
		return http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip"
	case errors.Is(err, domainTrip.ErrTripCompleted):
		return http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip"
	}

	var validationErr domainTrip.ValidationError
	if errors.As(err, &validationErr) {
		return http.StatusBadRequest, "invalid_request", "Invalid request", validationErr.Error()
	}

	return http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error"
}

func write(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status), //nolint:gosec // HTTP status codes always fit into int32.
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
}
