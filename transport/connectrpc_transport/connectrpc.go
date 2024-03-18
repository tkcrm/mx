package connectrpc_transport

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

const (
	defaultServiceName   = "connectrpc-server"
	defaultServerAddress = ":9000"
)

type connectRPCServer struct {
	Config

	name                 string
	server               *http.ServeMux
	logger               logger.Logger
	services             []ConnectRPCService
	serverHandlerWrapper func(http.Handler) http.Handler
	reflector            *grpcreflect.Reflector
	connectrpcOpts       []connect.HandlerOption
}

// NewServer creates a new gRPC server that implements service.IService interface.
func NewServer(opts ...Option) service.IService {
	srv := &connectRPCServer{
		name:   defaultServiceName,
		logger: logger.Default(),
		server: http.NewServeMux(),

		Config: Config{
			Enabled: true,
			Addr:    defaultServerAddress,
		},
	}

	for _, o := range opts {
		o(srv)
	}

	srv.logger = logger.With(srv.logger, "service", srv.name)

	for i := range srv.services {
		if srv.services[i] == nil {
			srv.logger.Errorf("empty connectrpc service #%d", i)
			continue
		}

		srv.logger.Infof("register connectrpc service: %s", srv.services[i].Name())

		path, handler := srv.services[i].RegisterHandler(srv.connectrpcOpts...)
		srv.server.Handle(path, handler)
	}

	if srv.reflector != nil {
		srv.server.Handle(grpcreflect.NewHandlerV1(srv.reflector))
		srv.server.Handle(grpcreflect.NewHandlerV1Alpha(srv.reflector))
	}

	return srv
}

// Name returns name of server.
func (s *connectRPCServer) Name() string { return s.name }

// Enabled returns is service enabled.
func (s *connectRPCServer) Enabled() bool { return s.Config.Enabled }

// Start allows starting server.
func (s *connectRPCServer) Start(ctx context.Context) error {
	s.logger.Infof("prepare listener %s on %s", s.name, s.Addr)

	var handler http.Handler = s.server
	if s.serverHandlerWrapper != nil {
		handler = s.serverHandlerWrapper(handler)
	}

	errChan := make(chan error, 1)
	go func() {
		if err := http.ListenAndServe(s.Addr, handler); err != nil {
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

// Stop allows to stop server.
func (s *connectRPCServer) Stop(context.Context) error { return nil }
