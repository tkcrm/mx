package grpc_transport

import (
	"context"
	"net"

	gprom "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

const (
	defaultGRPCName    = "grpc-server"
	defaultGRPCAddress = ":9000"
	defaultGRPCNetwork = "tcp"
)

type gRPCServer struct {
	Config

	name     string
	server   *grpc.Server
	logger   logger.Logger
	services []GRPCService
}

func defaultGRPCServer() *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(gprom.UnaryServerInterceptor, otelgrpc.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(gprom.StreamServerInterceptor, otelgrpc.StreamServerInterceptor()),
	)
}

// New creates new gRPC server and implements service.IService interface.
func New(opts ...Option) service.IService {
	srv := &gRPCServer{
		name:   defaultGRPCName,
		logger: logger.Default(),
		server: defaultGRPCServer(),

		Config: Config{
			Enabled: true,
			Addr:    defaultGRPCAddress,
			Network: defaultGRPCNetwork,
		},
	}

	for _, o := range opts {
		o(srv)
	}

	if srv.Reflect {
		srv.services = append(srv.services, new(reflectionService))
	}

	for i := range srv.services {
		if srv.services[i] == nil {
			srv.logger.Errorf("empty gRPC service #%d", i)

			continue
		}

		srv.logger.Infof("register gRPC service: %s", srv.services[i].Name())

		srv.services[i].Register(srv.server)
	}

	return srv
}

// Name returns name of gRPC server.
func (s *gRPCServer) Name() string { return s.name }

// Enabled returns is service enabled.
func (s *gRPCServer) Enabled() bool { return s.Config.Enabled }

// Start allows starting gRPC server.
func (s *gRPCServer) Start(ctx context.Context) error {
	s.logger.Infow("prepare listener",
		"name", s.name,
		"address", s.Addr,
		"network", s.Network)

	lis, err := new(net.ListenConfig).Listen(ctx, s.Network, s.Addr)
	if err != nil {
		return err
	}

	var errChan = make(chan error, 1)
	go func() {
		if err := s.server.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
	}

	return nil
}

// Stop allows to stop grpc server.
func (s *gRPCServer) Stop(context.Context) error {
	if s.server == nil {
		return nil
	}

	s.server.GracefulStop()

	return nil
}
