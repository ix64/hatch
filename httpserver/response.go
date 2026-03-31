package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func Adapt(handler HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			WriteError(w, r, zap.NewNop(), err)
		}
	})
}

func AdaptWithLogger(logger *zap.Logger, handler HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			WriteError(w, r, logger, err)
		}
	})
}

func WriteJSON(w http.ResponseWriter, status int, body any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, r *http.Request, logger *zap.Logger, err error) {
	prob := transformUnexpectedError(err)
	writeProblem(w, r, prob)

	if logger == nil {
		return
	}
	log := logger.With(zap.String("request_id", r.Header.Get(HeaderRequestID)))
	if ce := log.Check(prob.GetLogLevel(), "unexpected error"); ce != nil {
		ce.Write(prob.GetLogFields()...)
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, problem *Problem) {
	body := *problem
	if body.Instance == "" && r != nil && r.URL != nil {
		body.Instance = r.URL.Path
	}
	w.Header().Set("Content-Type", MIMEProblemJSON)
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(&body)
}

func transformUnexpectedError(oriErr error) *Problem {
	var prob *Problem
	if errors.As(oriErr, &prob) {
		return prob
	}

	var statusError interface{ StatusCode() int }
	ret := NewProblemException(http.StatusInternalServerError, "unexpected error").
		SetCode(CodeInternal).
		SetCause(oriErr)

	if errors.As(oriErr, &statusError) {
		status := statusError.StatusCode()
		if status >= 400 {
			ret.Status = status
			ret.SetCode(CodeRouteFailure)
			ret.Title = "route: " + http.StatusText(status)
			ret.SetLogLevel(zapcore.DebugLevel)
			ret.cause = nil
			return ret
		}
	}

	return ret
}

func BadRequest(detail string) *Problem {
	return NewProblemException(http.StatusBadRequest, http.StatusText(http.StatusBadRequest)).
		SetCode(CodeBadRequest).
		AddDetail(detail)
}

func NotFound(detail string) *Problem {
	return NewProblemException(http.StatusNotFound, http.StatusText(http.StatusNotFound)).
		SetCode(CodeNotFound).
		AddDetail(detail)
}

func ServiceUnavailable(detail string) *Problem {
	return NewProblemException(http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable)).
		SetCode(CodeServiceUnavailable).
		AddDetail(detail)
}

type panicStatusError struct{}

func (panicStatusError) Error() string   { return "panic recovered" }
func (panicStatusError) StatusCode() int { return http.StatusInternalServerError }
func (panicStatusError) Unwrap() error   { return nil }
func stacktrace() string                 { return string(debug.Stack()) }
func recoverProblem(v any) error {
	return fmt.Errorf("%w: %v\n%s", panicStatusError{}, v, stacktrace())
}
