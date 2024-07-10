package grpc_transport

import (
	"context"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/tkcrm/mx/logger"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	defaultGRPCName    = "grpc-server"
	defaultGRPCAddress = ":9000"
	defaultGRPCNetwork = "tcp"
)

type GRPCServer struct {
	Config

	name     string
	server   *grpc.Server
	logger   logger.Logger
	services []GRPCService
}

// NewServer creates a new gRPC server that implements service.IService interface.
func NewServer(opts ...Option) *GRPCServer {
	srv := &GRPCServer{
		name:   defaultGRPCName,
		logger: logger.Default(),

		Config: Config{
			Enabled: true,
			Addr:    defaultGRPCAddress,
			Network: defaultGRPCNetwork,
		},
	}

	for _, o := range opts {
		o(srv)
	}

	srv.logger = logger.With(srv.logger, "service", srv.name)

	if srv.server == nil {
		// define unary interceptors
		unaryInterceptors := []grpc.UnaryServerInterceptor{}

		// define stream interceptors
		streamInterceptors := []grpc.StreamServerInterceptor{}

		// add logger
		if srv.LoggerEnabled {
			unaryInterceptors = append(unaryInterceptors,
				logging.UnaryServerInterceptor(InterceptorLogger(srv.logger)),
			)
			streamInterceptors = append(streamInterceptors,
				logging.StreamServerInterceptor(InterceptorLogger(srv.logger)),
			)
		}

		// add recovery
		if srv.RecoveryEnabled {
			opts := []recovery.Option{
				recovery.WithRecoveryHandler(RecoveryFunc(srv.logger)),
			}
			unaryInterceptors = append(unaryInterceptors, recovery.UnaryServerInterceptor(opts...))
			streamInterceptors = append(streamInterceptors, recovery.StreamServerInterceptor(opts...))
		}

		// define grpc server options
		srvOpts := []grpc.ServerOption{
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
			grpc.ChainUnaryInterceptor(unaryInterceptors...),
			grpc.ChainStreamInterceptor(streamInterceptors...),
		}

		// init default grpc server
		srv.server = grpc.NewServer(srvOpts...)
	}

	if srv.ReflectEnabled {
		srv.services = append(srv.services, new(reflectionService))
	}

	if srv.HealthCheckEnabled {
		grpc_health_v1.RegisterHealthServer(srv.server, health.NewServer())
	}

	for i := range srv.services {
		if srv.services[i] == nil {
			srv.logger.Errorf("empty grpc service #%d", i)
			continue
		}

		srv.logger.Infof("register grpc service: %s", srv.services[i].Name())

		srv.services[i].Register(srv.server)
	}

	return srv
}

// Name returns name of gRPC server.
func (s *GRPCServer) Name() string { return s.name }

// Enabled returns is service enabled.
func (s *GRPCServer) Enabled() bool { return s.Config.Enabled }

// Start allows starting gRPC server.
func (s *GRPCServer) Start(ctx context.Context) error {
	s.logger.Infof("prepare listener %s on %s / %s",
		s.name, s.Addr, s.Network,
	)

	lis, err := new(net.ListenConfig).Listen(ctx, s.Network, s.Addr)
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.server.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return nil
	}
}

// Stop allows to stop grpc server.
func (s *GRPCServer) Stop(context.Context) error {
	if s.server == nil {
		return nil
	}

	s.server.GracefulStop()

	return nil
}
