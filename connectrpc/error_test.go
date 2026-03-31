package connectrpc

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
)

type statusErr struct{}

func (statusErr) Error() string          { return "vm not found" }
func (statusErr) StatusCode() int        { return 404 }
func (statusErr) ConnectMessage() string { return "vm not found" }

func TestAsErrorFromStatusCodeError(t *testing.T) {
	t.Parallel()

	err := AsError(statusErr{})
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("AsError() error type = %T", err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("AsError() code = %v", connectErr.Code())
	}
	if connectErr.Message() != "vm not found" {
		t.Fatalf("AsError() message = %q", connectErr.Message())
	}
}

func TestAsErrorFallsBackToErrorMessage(t *testing.T) {
	t.Parallel()

	err := AsError(errors.New("boom"))
	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("AsError() error type = %T", err)
	}
	if connectErr.Code() != connect.CodeInternal {
		t.Fatalf("AsError() code = %v", connectErr.Code())
	}
	if connectErr.Message() != "boom" {
		t.Fatalf("AsError() message = %q", connectErr.Message())
	}
}
