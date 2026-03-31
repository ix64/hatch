package httpserver

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type lifecycleRecorder struct {
	hooks []fx.Hook
}

func (l *lifecycleRecorder) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func TestRunServerReturnsStartError(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	lc := &lifecycleRecorder{}
	RunServer(lc, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Config{
		Addr: listener.Addr().String(),
	}, zap.NewNop())

	if len(lc.hooks) != 1 {
		t.Fatalf("expected exactly one lifecycle hook, got %d", len(lc.hooks))
	}

	err = lc.hooks[0].OnStart(context.Background())
	if err == nil {
		t.Fatal("expected start error when address is already in use")
	}
	if !strings.Contains(err.Error(), "bind") && !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected bind/listen error, got %v", err)
	}
}

func TestRunServerServesAndStops(t *testing.T) {
	lc := &lifecycleRecorder{}
	logger := zap.NewNop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	RunServer(lc, handler, Config{Addr: "127.0.0.1:0"}, logger)
	if len(lc.hooks) != 1 {
		t.Fatalf("expected exactly one lifecycle hook, got %d", len(lc.hooks))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := lc.hooks[0].OnStart(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	if err := lc.hooks[0].OnStop(ctx); err != nil {
		t.Fatalf("stop server: %v", err)
	}
}
