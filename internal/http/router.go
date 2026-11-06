package http

import (
	nethttp "net/http"

	"github.com/a-pelikh/tripgo/api"
	"github.com/a-pelikh/tripgo/internal/http/domainerror"
	"github.com/a-pelikh/tripgo/internal/http/health"
	"github.com/a-pelikh/tripgo/internal/http/ready"
	tripCreate "github.com/a-pelikh/tripgo/internal/http/trip/create"
	tripFinish "github.com/a-pelikh/tripgo/internal/http/trip/finish"
	tripGet "github.com/a-pelikh/tripgo/internal/http/trip/get"
)

type Router struct {
	createTrip *tripCreate.Handler
	getTrip    *tripGet.Handler
	finishTrip *tripFinish.Handler
	health     *health.Handler
	ready      *ready.Handler
}

func NewRouter(
	createTrip *tripCreate.Handler,
	getTrip *tripGet.Handler,
	finishTrip *tripFinish.Handler,
	healthHandler *health.Handler,
	readyHandler *ready.Handler,
) nethttp.Handler {
	return api.HandlerWithOptions(
		&Router{
			createTrip: createTrip,
			getTrip:    getTrip,
			finishTrip: finishTrip,
			health:     healthHandler,
			ready:      readyHandler,
		},
		api.ChiServerOptions{ErrorHandlerFunc: bindingError},
	)
}

func (r *Router) CreateTrip(w nethttp.ResponseWriter, request *nethttp.Request, params api.CreateTripParams) {
	r.createTrip.CreateTrip(w, request, params)
}

func (r *Router) GetTrip(w nethttp.ResponseWriter, request *nethttp.Request, tripID api.TripId) {
	r.getTrip.GetTrip(w, request, tripID)
}

func (r *Router) FinishTrip(w nethttp.ResponseWriter, request *nethttp.Request, tripID api.TripId) {
	r.finishTrip.FinishTrip(w, request, tripID)
}

func (r *Router) Health(w nethttp.ResponseWriter, request *nethttp.Request) {
	r.health.Health(w, request)
}

func (r *Router) Ready(w nethttp.ResponseWriter, request *nethttp.Request) {
	r.ready.Ready(w, request)
}

func bindingError(w nethttp.ResponseWriter, r *nethttp.Request, _ error) {
	domainerror.InvalidRequest(w, r, "Request validation failed")
}
