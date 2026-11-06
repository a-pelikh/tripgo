package middleware

import (
	"context"
	"net/http"

	"github.com/a-pelikh/tripgo/internal/http/problem"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/nethttp-middleware"
)

func Validate(spec *openapi3.T, handler http.Handler) http.Handler {
	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(
		spec,
		&nethttpmiddleware.Options{
			DoNotValidateServers: true,
			ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ nethttpmiddleware.ErrorHandlerOpts) {
				problem.Write(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
			},
		},
	)

	return validator(handler)
}
