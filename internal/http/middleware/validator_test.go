package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-pelikh/tripgo/api"
	"github.com/a-pelikh/tripgo/internal/http/middleware"
)

func TestValidateRejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	spec, err := api.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}

	called := false
	handler := middleware.Validate(spec, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if called {
		t.Fatal("invalid request reached the next handler")
	}
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", contentType)
	}
	if !strings.Contains(response.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("response body = %s, want invalid_request problem", response.Body.String())
	}
}
