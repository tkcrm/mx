package grpc_transport

import (
	"runtime/debug"

	"github.com/tkcrm/mx/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RecoveryFunc(l logger.Logger) func(p any) (err error) {
	return func(p any) (err error) {
		logger.With(l, "stack", string(debug.Stack())).Errorf("recovered from panic: %v", p)
		return status.Errorf(codes.Internal, "recovered from panic: %v", p)
	}
}
