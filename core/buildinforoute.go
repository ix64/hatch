package core

import (
	"encoding/json"
	"net/http"

	"github.com/ix64/hatch/httpserver"
)

type buildInfoRoute struct{}

func NewBuildInfoRoute() httpserver.Route {
	return &buildInfoRoute{}
}

func (r *buildInfoRoute) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, req *http.Request) {
		info := CurrentBuildInfo()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"module":      info.Module,
			"version":     info.Version,
			"commit_hash": info.CommitHash,
			"build_time":  info.BuildTime,
		})
	})
}
