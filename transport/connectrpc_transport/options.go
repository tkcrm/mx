package connectrpc_transport

import (
	"net/http"

	"connectrpc.com/grpcreflect"
	"github.com/tkcrm/mx/logger"
)

// Option allows customizing gRPC server.
type Option func(s *connectRPCServer)

// WithName allows set custom gRPC Name.
func WithName(v string) Option {
	return func(s *connectRPCServer) {
		if v == "" {
			return
		}
		s.name = v
	}
}

// WithConfig allows set custom gRPC settings.
func WithConfig(v Config) Option {
	return func(s *connectRPCServer) { s.Config = v }
}

// WithLogger allows set custom gRPC Logger.
func WithLogger(v logger.Logger) Option {
	return func(s *connectRPCServer) {
		if v == nil {
			return
		}
		s.logger = v
	}
}

// WithServer allows set custom gRPC Server.
func WithServer(v *http.ServeMux) Option {
	return func(s *connectRPCServer) {
		if v == nil {
			return
		}
		s.server = v
	}
}

// WithServer allows to use reflection for grpc server.
func WithReflector(v *grpcreflect.Reflector) Option {
	return func(s *connectRPCServer) {
		s.reflector = v
	}
}

// WithServerHandlerWrapper allows set custom server handler wrapper.
func WithServerHandlerWrapper(v func() http.Handler) Option {
	return func(s *connectRPCServer) {
		if v == nil {
			return
		}
		s.serverHandlerWrapper = v
	}
}

// WithServices allows adding new gRPC Service.
func WithServices(services ...ConnectRPCService) Option {
	return func(s *connectRPCServer) { s.services = append(s.services, services...) }
}
