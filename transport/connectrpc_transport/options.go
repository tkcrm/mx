package connectrpc_transport

import (
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/tkcrm/mx/logger"
)

// Option allows customizing gRPC server.
type Option func(s *ConnectRPCServer)

// WithName allows set custom gRPC Name.
func WithName(v string) Option {
	return func(s *ConnectRPCServer) {
		if v == "" {
			return
		}
		s.name = v
	}
}

// WithConfig allows set custom gRPC settings.
func WithConfig(v Config) Option {
	return func(s *ConnectRPCServer) { s.Config = v }
}

// WithLogger allows set custom gRPC Logger.
func WithLogger(v logger.Logger) Option {
	return func(s *ConnectRPCServer) {
		if v == nil {
			return
		}
		s.logger = v
	}
}

// WithServer allows set custom gRPC Server.
func WithServeMux(v *http.ServeMux) Option {
	return func(s *ConnectRPCServer) {
		if v == nil {
			return
		}
		s.serveMux = v
	}
}

// WithHttpServer allows set custom http.Server.
func WithHttpServer(v *http.Server) Option {
	return func(s *ConnectRPCServer) {
		if s == nil {
			return
		}
		s.httpServer = v
	}
}

// WithServer allows to use reflection for grpc server.
func WithReflection(services ...string) Option {
	return func(s *ConnectRPCServer) {
		s.reflector = grpcreflect.NewStaticReflector(services...)
	}
}

// WithServerHandlerWrapper allows set custom server handler wrapper.
func WithServerHandlerWrapper(v func(h http.Handler) http.Handler) Option {
	return func(s *ConnectRPCServer) {
		if v == nil {
			return
		}
		s.serverHandlerWrapper = v
	}
}

// WithServices allows adding new gRPC Service.
func WithServices(services ...ConnectRPCService) Option {
	return func(s *ConnectRPCServer) { s.services = append(s.services, services...) }
}

// WithConnectRPCOptions allows adding new connect rpc options.
func WithConnectRPCOptions(opts ...connect.HandlerOption) Option {
	return func(s *ConnectRPCServer) {
		s.connectrpcOpts = append(s.connectrpcOpts, opts...)
	}
}
