package health

import (
	"context"
	"encoding/json"
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
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		for _, probe := range r.probes {
			if err := probe.Check(ctx); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{
					"status": "not_ready",
					"error":  probe.Name() + ": " + err.Error(),
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
