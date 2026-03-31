package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set(HeaderRequestID, requestID)
		r.Header.Set(HeaderRequestID, requestID)
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				WriteError(w, r, logger, recoverProblem(v))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func accessLogMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(recorder, r)
		logger.Info("access",
			zap.String("method", r.Method),
			zap.String("uri", r.URL.RequestURI()),
			zap.String("request_id", recorder.Header().Get(HeaderRequestID)),
			zap.String("remote_ip", r.RemoteAddr),
			zap.Int("status", recorder.status),
			zap.Duration("duration", time.Since(started)),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-User-ID, X-Tenant-ID, X-User-Role")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
