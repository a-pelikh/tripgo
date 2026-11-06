package create_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/a-pelikh/tripgo/api"
	domainTrip "github.com/a-pelikh/tripgo/internal/domain/trip"
	tripCreate "github.com/a-pelikh/tripgo/internal/http/trip/create"
)

const validRequest = `{
  "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
  "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
  "start_point":{"latitude":59.9398,"longitude":30.3146},
  "end_point":{"latitude":59.929,"longitude":30.3626},
  "price":1450
}`

func TestHandlerCreateTripSuccess(t *testing.T) {
	t.Parallel()

	tripEntity := newTrip(t)
	ctrl := gomock.NewController(t)
	creator := NewMockuseCase(ctrl)
	creator.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), int64(1450)).
		Return(tripEntity, nil)
	handler := tripCreate.New(discardLogger(), creator)
	response := httptest.NewRecorder()

	handler.CreateTrip(response, createRequest(validRequest), api.CreateTripParams{})

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if got, want := response.Header().Get("Location"), "/api/v1/trips/"+uuid.UUID(tripEntity.ID()).String(); got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}
}

func TestHandlerCreateTripErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		useCaseErr error
		wantStatus int
		wantCode   string
	}{
		{name: "driver busy", useCaseErr: domainTrip.ErrDriverBusy, wantStatus: http.StatusConflict, wantCode: "driver_busy"},
		{name: "internal error", useCaseErr: errors.New("database secret"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			creator := NewMockuseCase(ctrl)
			creator.EXPECT().
				Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), int64(1450)).
				Return(domainTrip.Trip{}, tt.useCaseErr)
			handler := tripCreate.New(discardLogger(), creator)
			response := httptest.NewRecorder()

			handler.CreateTrip(response, createRequest(validRequest), api.CreateTripParams{})

			assertProblem(t, response, tt.wantStatus, tt.wantCode)
			if strings.Contains(response.Body.String(), tt.useCaseErr.Error()) {
				t.Fatal("response contains internal error details")
			}
		})
	}
}

func TestHandlerCreateTripInvalidRequest(t *testing.T) {
	t.Parallel()

	tests := []string{
		`{`,
		`{"user_id":"00000000-0000-0000-0000-000000000000","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59,"longitude":30},"end_point":{"latitude":59,"longitude":30},"price":1450}`,
		`{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":91,"longitude":30},"end_point":{"latitude":59,"longitude":30},"price":1450}`,
	}

	for _, body := range tests {
		body := body
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			handler := tripCreate.New(discardLogger(), NewMockuseCase(ctrl))
			response := httptest.NewRecorder()

			handler.CreateTrip(response, createRequest(body), api.CreateTripParams{})

			assertProblem(t, response, http.StatusBadRequest, "invalid_request")
		})
	}
}

func newTrip(t *testing.T) domainTrip.Trip {
	t.Helper()

	startPoint, err := domainTrip.NewCoordinates(59.9398, 30.3146)
	if err != nil {
		t.Fatalf("create start point: %v", err)
	}
	endPoint, err := domainTrip.NewCoordinates(59.929, 30.3626)
	if err != nil {
		t.Fatalf("create end point: %v", err)
	}
	tripEntity, err := domainTrip.New(
		domainTrip.ID(uuid.New()),
		domainTrip.UserID(uuid.New()),
		domainTrip.DriverID(uuid.New()),
		startPoint,
		endPoint,
		1450,
		domainTrip.StatusActive,
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		t.Fatalf("create trip: %v", err)
	}

	return tripEntity
}

func createRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(body))
}

func assertProblem(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	if response.Code != status {
		t.Fatalf("status = %d, want %d", response.Code, status)
	}
	if !strings.Contains(response.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("response body = %s, want problem code %q", response.Body.String(), code)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
