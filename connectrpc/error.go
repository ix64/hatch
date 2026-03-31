package connectrpc

import (
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/ix64/hatch/httpserver"
)

func ValidateID(id int64, name string) (int64, error) {
	if id <= 0 {
		return 0, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid %s", name))
	}
	return id, nil
}

func AsError(err error) error {
	if err == nil {
		return nil
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}

	var problem *httpserver.Problem
	if errors.As(err, &problem) {
		return connect.NewError(codeFromHTTPStatus(problem.Status), errors.New(problemMessage(problem)))
	}

	return connect.NewError(connect.CodeInternal, err)
}

func codeFromHTTPStatus(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAlreadyExists
	case http.StatusUnprocessableEntity:
		return connect.CodeFailedPrecondition
	case http.StatusNotImplemented:
		return connect.CodeUnimplemented
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	case http.StatusGatewayTimeout:
		return connect.CodeDeadlineExceeded
	default:
		if status >= 500 {
			return connect.CodeInternal
		}
		return connect.CodeUnknown
	}
}

func problemMessage(problem *httpserver.Problem) string {
	if problem == nil {
		return "unexpected error"
	}
	if problem.Detail != "" {
		return problem.Detail
	}
	if problem.Title != "" {
		return problem.Title
	}
	return "unexpected error"
}
