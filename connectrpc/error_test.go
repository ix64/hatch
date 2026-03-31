package connectrpc

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/ix64/hatch/httpserver"
)

func TestAsErrorFromProblem(t *testing.T) {
	t.Parallel()

	err := AsError(httpserver.NotFound("vm not found"))
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
