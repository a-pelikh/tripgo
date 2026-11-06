package get_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	tripGet "github.com/a-pelikh/tripgo/internal/http/trip/get"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestHandlerGetTripSuccess(t *testing.T) {
	t.Parallel()

	tripEntity := newTrip(t)
	ctrl := gomock.NewController(t)
	getter := NewMockuseCase(ctrl)
	getter.EXPECT().Execute(gomock.Any(), tripEntity.ID()).Return(tripEntity, nil)
	response := httptest.NewRecorder()

	tripGet.New(discardLogger(), getter).GetTrip(response, request(), uuid.UUID(tripEntity.ID()))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), uuid.UUID(tripEntity.ID()).String()) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestHandlerGetTripErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not found", err: domainTrip.ErrTripNotFound, status: http.StatusNotFound, code: "trip_not_found"},
		{name: "internal", err: errors.New("database secret"), status: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tripID := uuid.New()
			ctrl := gomock.NewController(t)
			getter := NewMockuseCase(ctrl)
			getter.EXPECT().Execute(gomock.Any(), domainTrip.ID(tripID)).Return(domainTrip.Trip{}, tt.err)
			response := httptest.NewRecorder()

			tripGet.New(discardLogger(), getter).GetTrip(response, request(), tripID)

			assertProblem(t, response, tt.status, tt.code)
			if strings.Contains(response.Body.String(), tt.err.Error()) {
				t.Fatal("response contains internal error details")
			}
		})
	}
}

func TestHandlerGetTripRejectsEmptyID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	response := httptest.NewRecorder()
	tripGet.New(discardLogger(), NewMockuseCase(ctrl)).GetTrip(response, request(), uuid.Nil)
	assertProblem(t, response, http.StatusBadRequest, "invalid_request")
}

func newTrip(t *testing.T) domainTrip.Trip {
	t.Helper()

	start, _ := domainTrip.NewCoordinates(59.9, 30.3)
	end, _ := domainTrip.NewCoordinates(59.8, 30.4)
	entity, err := domainTrip.New(
		domainTrip.ID(uuid.New()),
		domainTrip.UserID(uuid.New()),
		domainTrip.DriverID(uuid.New()),
		start,
		end,
		1450,
		domainTrip.StatusActive,
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}
	return entity
}

func request() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/api/v1/trips/id", nil)
}

func assertProblem(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status || !strings.Contains(response.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("response = %d %s, want status %d code %s", response.Code, response.Body.String(), status, code)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
