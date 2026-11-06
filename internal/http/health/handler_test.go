package health_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-pelikh/tripgo/internal/http/health"
)

func TestHandlerHealth(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	health.New().Health(response, httptest.NewRequest(http.MethodGet, "/health", nil))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"ok"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
