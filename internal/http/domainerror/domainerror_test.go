package domainerror_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	"github.com/a-pelikh/tripgo/internal/http/domainerror"
)

func TestWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not found", err: domainTrip.ErrTripNotFound, status: http.StatusNotFound, code: "trip_not_found"},
		{name: "driver busy", err: domainTrip.ErrDriverBusy, status: http.StatusConflict, code: "driver_busy"},
		{name: "completed", err: domainTrip.ErrTripCompleted, status: http.StatusConflict, code: "trip_completed"},
		{name: "validation", err: domainTrip.ValidationError{Field: "price", Reason: "must not be negative"}, status: http.StatusBadRequest, code: "invalid_request"},
		{name: "internal", err: errors.New("database secret"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/id", nil)

			status := domainerror.Write(response, request, tt.err)

			if status != tt.status || response.Code != tt.status {
				t.Fatalf("status = %d/%d, want %d", status, response.Code, tt.status)
			}
			if !strings.Contains(response.Body.String(), `"code":"`+tt.code+`"`) {
				t.Fatalf("body = %s, want code %q", response.Body.String(), tt.code)
			}
			if tt.status == http.StatusInternalServerError && strings.Contains(response.Body.String(), tt.err.Error()) {
				t.Fatal("response contains internal error details")
			}
		})
	}
}
