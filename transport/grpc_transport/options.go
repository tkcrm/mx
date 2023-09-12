package grpc_transport

import (
	"github.com/tkcrm/mx/logger"
	"google.golang.org/grpc"
)

// Option allows customizing gRPC server.
type Option func(s *gRPCServer)

// GRPCService custom interface for gRPC service.
type GRPCService interface {
	Name() string
	Register(server *grpc.Server)
}

// WithName allows set custom gRPC Name.
func WithName(v string) Option {
	return func(s *gRPCServer) { s.name = v }
}

// WithConfig allows set custom gRPC settings.
func WithConfig(v Config) Option {
	return func(s *gRPCServer) { s.Config = v }
}

// WithLogger allows set custom gRPC Logger.
func WithLogger(v logger.Logger) Option {
	return func(s *gRPCServer) { s.logger = v }
}

// WithServer allows set custom gRPC Server.
func WithServer(v *grpc.Server) Option {
	return func(s *gRPCServer) { s.server = v }
}

// WithServices allows adding new gRPC Service.
func WithServices(services ...GRPCService) Option {
	return func(s *gRPCServer) { s.services = append(s.services, services...) }
}
