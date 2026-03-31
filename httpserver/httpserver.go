package httpserver

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Addr  string
	Debug bool
}

type Route interface {
	Register(mux *http.ServeMux)
}

type RouteParams struct {
	fx.In

	Mux    *http.ServeMux
	Routes []Route `group:"http_routes"`
}

func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"http_routes"`),
	)
}

func NewMux() *http.ServeMux {
	return http.NewServeMux()
}

func RegisterRoutes(params RouteParams) {
	for _, route := range params.Routes {
		route.Register(params.Mux)
	}
}

func NewHandler(mux *http.ServeMux) http.Handler {
	return mux
}

func RunServer(lc fx.Lifecycle, handler http.Handler, cfg Config, logger *zap.Logger) {
	addr := cfg.Addr
	if addr == "" {
		addr = ":9580"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			go func() {
				displayAddr := addr
				if strings.HasPrefix(displayAddr, ":") {
					displayAddr = "localhost" + displayAddr
				}
				logger.Named("server").Info("starting server", zap.String("address", displayAddr))
				if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
					logger.Named("server").Error("failed to start server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
