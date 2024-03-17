package connectrpc_transport

import "net/http"

// IService custom interface for gRPC service.
type ConnectRPCService interface {
	Name() string
	RegisterHandler() (string, http.Handler)
}
