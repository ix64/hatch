package health

import (
	"context"
	"net/http"

	"go.uber.org/fx"

	"github.com/ix64/hatch/httpserver"
)

type Probe interface {
	Name() string
	Check(ctx context.Context) error
}

type probeRoute struct {
	probes []Probe
}

type probeParams struct {
	fx.In

	Probes []Probe `group:"health_probes"`
}

func AsProbe(f any) any {
	return fx.Annotate(f, fx.As(new(Probe)), fx.ResultTags(`group:"health_probes"`))
}

func NewRoute(params probeParams) httpserver.Route {
	return &probeRoute{probes: params.Probes}
}

func (r *probeRoute) Register(mux *http.ServeMux) {
	mux.Handle("GET /healthz", httpserver.Adapt(func(w http.ResponseWriter, req *http.Request) error {
		return httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	mux.Handle("GET /readyz", httpserver.Adapt(func(w http.ResponseWriter, req *http.Request) error {
		ctx := req.Context()
		for _, probe := range r.probes {
			if err := probe.Check(ctx); err != nil {
				return httpserver.ServiceUnavailable(probe.Name() + ": " + err.Error())
			}
		}
		return httpserver.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}))
}
