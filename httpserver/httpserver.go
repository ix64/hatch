package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

const HeaderRequestID = "X-Request-ID"

type Config struct {
	Addr  string
	Debug bool
}

type Route interface {
	Register(mux *http.ServeMux)
}

type Middleware func(http.Handler) http.Handler

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

func NewHandler(
	mux *http.ServeMux,
	logger *zap.Logger,
	stdLogger *slog.Logger,
	cfg Config,
	authMiddleware func(http.Handler) http.Handler,
) http.Handler {
	handler := http.Handler(mux)
	handler = requestIDMiddleware(handler)
	handler = recoverMiddleware(logger, handler)
	if authMiddleware != nil {
		handler = authMiddleware(handler)
	}
	handler = accessLogMiddleware(logger.Named("http"), handler)
	if cfg.Debug {
		handler = corsMiddleware(handler)
	}

	if stdLogger != nil {
		_ = stdLogger
	}

	return handler
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
			go func() {
				displayAddr := addr
				if strings.HasPrefix(displayAddr, ":") {
					displayAddr = "localhost" + displayAddr
				}
				logger.Named("server").Info("starting server", zap.String("address", displayAddr))
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
