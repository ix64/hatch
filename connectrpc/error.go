package connectrpc

import (
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
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

	var statusErr interface{ StatusCode() int }
	if errors.As(err, &statusErr) {
		return connect.NewError(codeFromHTTPStatus(statusErr.StatusCode()), errors.New(errorMessage(err)))
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

func errorMessage(err error) string {
	if err == nil {
		return "unexpected error"
	}
	var carrier interface{ ConnectMessage() string }
	if errors.As(err, &carrier) {
		return carrier.ConnectMessage()
	}
	return err.Error()
}
