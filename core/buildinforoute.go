package core

import (
	"net/http"

	"github.com/ix64/hatch/httpserver"
)

type buildInfoRoute struct{}

func NewBuildInfoRoute() httpserver.Route {
	return &buildInfoRoute{}
}

func (r *buildInfoRoute) Register(mux *http.ServeMux) {
	mux.Handle("GET /version", httpserver.Adapt(func(w http.ResponseWriter, req *http.Request) error {
		info := CurrentBuildInfo()

		return httpserver.WriteJSON(w, http.StatusOK, map[string]string{
			"module":      info.Module,
			"version":     info.Version,
			"commit_hash": info.CommitHash,
			"build_time":  info.BuildTime,
		})
	}))
}
