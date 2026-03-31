package connectrpc

import (
	"net/http"

	"connectrpc.com/grpcreflect"
	"go.uber.org/fx"
)

type Handler interface {
	Register(mux *http.ServeMux) []string
}

type handlerParams struct {
	fx.In

	Mux      *http.ServeMux
	Handlers []Handler `group:"connect_handlers"`
}

func AsHandler(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Handler)),
		fx.ResultTags(`group:"connect_handlers"`),
	)
}

func RegisterHandlers(params handlerParams) {
	services := make([]string, 0, len(params.Handlers))
	for _, handler := range params.Handlers {
		services = append(services, handler.Register(params.Mux)...)
	}

	reflector := grpcreflect.NewStaticReflector(services...)
	v1Prefix, v1Handler := grpcreflect.NewHandlerV1(reflector)
	params.Mux.Handle(v1Prefix, v1Handler)
	alphaPrefix, alphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	params.Mux.Handle(alphaPrefix, alphaHandler)
}
