package ready_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/a-pelikh/tripgo/internal/http/ready"
)

func TestHandlerReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{name: "ready", status: http.StatusOK, body: `"status":"ok"`},
		{name: "unavailable", err: errors.New("database unavailable"), status: http.StatusServiceUnavailable, body: `"status":"unavailable"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			checker := NewMockreadinessChecker(ctrl)
			checker.EXPECT().Ping(gomock.Any()).Return(tt.err)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/ready", nil)

			ready.New(checker).Ready(response, request)

			if response.Code != tt.status || !strings.Contains(response.Body.String(), tt.body) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}
