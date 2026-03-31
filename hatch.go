package hatch

import "go.uber.org/fx"

// Bootstrap creates a new Fx application with the provided options.
func Bootstrap(opts ...fx.Option) *fx.App {
	return fx.New(opts...)
}
