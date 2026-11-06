package middleware

import (
	"context"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/nethttp-middleware"

	"github.com/a-pelikh/tripgo/internal/http/domainerror"
)

func Validate(spec *openapi3.T, handler http.Handler) http.Handler {
	validator := nethttpmiddleware.OapiRequestValidatorWithOptions(
		spec,
		&nethttpmiddleware.Options{
			DoNotValidateServers: true,
			ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ nethttpmiddleware.ErrorHandlerOpts) {
				domainerror.InvalidRequest(w, r, "Request validation failed")
			},
		},
	)

	return validator(handler)
}
