package grpc_transport

import (
	"github.com/tkcrm/mx/logger"
	"google.golang.org/grpc"
)

// Option allows customizing gRPC server.
type Option func(s *GRPCServer)

// GRPCService custom interface for gRPC service.
type GRPCService interface {
	Name() string
	Register(server *grpc.Server)
}

// WithName allows set custom gRPC Name.
func WithName(v string) Option {
	return func(s *GRPCServer) {
		if v == "" {
			return
		}
		s.name = v
	}
}

// WithConfig allows set custom gRPC settings.
func WithConfig(v Config) Option {
	return func(s *GRPCServer) { s.Config = v }
}

// WithLogger allows set custom gRPC Logger.
func WithLogger(v logger.Logger) Option {
	return func(s *GRPCServer) {
		if v == nil {
			return
		}
		s.logger = v
	}
}

// WithServer allows set custom gRPC Server.
func WithServer(v *grpc.Server) Option {
	return func(s *GRPCServer) {
		if v == nil {
			return
		}
		s.server = v
	}
}

// WithServices allows adding new gRPC Service.
func WithServices(services ...GRPCService) Option {
	return func(s *GRPCServer) { s.services = append(s.services, services...) }
}
