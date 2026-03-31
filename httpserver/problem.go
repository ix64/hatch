package httpserver

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const MIMEProblemJSON = "application/problem+json"

const (
	CodeUnspecified        = "UNSPECIFIED"
	CodeNotFound           = "NOT_FOUND"
	CodeBadRequest         = "BAD_REQUEST"
	CodeInternal           = "INTERNAL_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeRouteFailure       = "ROUTE_FAILURE"
)

type Problem struct {
	Status   int    `json:"status"`
	Title    string `json:"title"`
	Instance string `json:"instance"`

	Code string `json:"code"`
	Type string `json:"type,omitempty"`

	Detail string `json:"detail,omitempty"`
	Extra  any    `json:"extra,omitempty"`

	cause      error
	stacktrace *zap.Field
	level      zapcore.Level
}

func (e *Problem) SetCode(code string) *Problem {
	e.Code = code
	e.Type = problemTypeBase + codeToSlug(code)
	return e
}

func (e *Problem) StatusCode() int {
	return e.Status
}

func (e *Problem) AddDetail(detail string) *Problem {
	if e.Detail != "" {
		e.Detail += "\n"
	}
	e.Detail += detail
	return e
}

func (e *Problem) SetExtra(extra any) *Problem {
	e.Extra = extra
	return e
}

func (e *Problem) SetCause(err error) *Problem {
	e.cause = err
	e.level = zapcore.ErrorLevel
	st := zap.StackSkip("stacktrace", 1)
	e.stacktrace = &st
	return e
}

func (e *Problem) SetLogLevel(level zapcore.Level) *Problem {
	e.level = level
	return e
}

func (e *Problem) GetLogLevel() zapcore.Level {
	return e.level
}

func (e *Problem) GetLogFields() []zap.Field {
	fields := []zap.Field{
		zap.Int("status", e.Status),
		zap.String("code", e.Code),
		zap.String("title", e.Title),
	}

	if e.Detail != "" {
		fields = append(fields, zap.String("detail", e.Detail))
	}
	if e.Extra != nil {
		fields = append(fields, zap.Any("extra", e.Extra))
	}
	if e.cause != nil {
		fields = append(fields, zap.NamedError("cause", e.cause))
	}
	if e.stacktrace != nil && e.level >= zapcore.WarnLevel {
		fields = append(fields, *e.stacktrace)
	}
	return fields
}

func (e *Problem) Error() string {
	return fmt.Sprintf("status: %d, title: %s", e.Status, e.Title)
}

func NewProblem(title string) *Problem {
	return &Problem{
		Status: http.StatusUnprocessableEntity,
		Title:  title,
		Code:   CodeUnspecified,
		Type:   problemTypeBase + codeToSlug(CodeUnspecified),
		level:  zapcore.InfoLevel,
	}
}

func NewProblemException(httpCode int, title string) *Problem {
	return &Problem{
		Status: httpCode,
		Title:  title,
		Code:   CodeUnspecified,
		Type:   problemTypeBase + codeToSlug(CodeUnspecified),
		level:  zapcore.InfoLevel,
	}
}

func WrapProblem(err error) *Problem {
	var prob *Problem
	if errors.As(err, &prob) {
		return prob
	}
	return NewProblem(err.Error())
}

const problemTypeBase = "https://sparkvm.github.io/docs/error-code/"

func codeToSlug(code string) string {
	if code == "" {
		return "error"
	}
	return strings.ToLower(strings.ReplaceAll(code, "_", "-"))
}
