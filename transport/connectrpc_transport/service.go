package connectrpc_transport

import (
	"net/http"

	"connectrpc.com/connect"
)

// IService custom interface for gRPC service.
type ConnectRPCService interface {
	Name() string
	RegisterHandler(otps ...connect.HandlerOption) (string, http.Handler)
}
