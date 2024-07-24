package connectrpc_transport

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/tkcrm/mx/logger"
)

const (
	defaultServiceName   = "connectrpc-server"
	defaultServerAddress = ":9000"
)

type ConnectRPCServer struct {
	Config

	name                 string
	httpServer           *http.Server
	serveMux             *http.ServeMux
	logger               logger.Logger
	services             []ConnectRPCService
	serverHandlerWrapper func(http.Handler) http.Handler
	reflector            *grpcreflect.Reflector
	connectrpcOpts       []connect.HandlerOption
}

// NewServer creates a new gRPC server that implements service.IService interface.
func NewServer(opts ...Option) *ConnectRPCServer {
	srv := &ConnectRPCServer{
		name:     defaultServiceName,
		logger:   logger.Default(),
		serveMux: http.NewServeMux(),

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
		srv.serveMux.Handle(path, handler)
	}

	if srv.reflector != nil {
		srv.serveMux.Handle(grpcreflect.NewHandlerV1(srv.reflector))
		srv.serveMux.Handle(grpcreflect.NewHandlerV1Alpha(srv.reflector))
	}

	return srv
}

// Name returns name of server.
func (s *ConnectRPCServer) Name() string { return s.name }

// Enabled returns is service enabled.
func (s *ConnectRPCServer) Enabled() bool { return s.Config.Enabled }

// Start allows starting server.
func (s *ConnectRPCServer) Start(ctx context.Context) error {
	s.logger.Infof("prepare listener %s on %s", s.name, s.Addr)

	var handler http.Handler = s.serveMux
	if s.serverHandlerWrapper != nil {
		handler = s.serverHandlerWrapper(handler)
	}

	if s.httpServer == nil {
		s.httpServer = &http.Server{
			Addr:              s.Addr,
			Handler:           handler,
			ReadHeaderTimeout: time.Second * 10,
		}
	} else {
		s.httpServer.Handler = handler
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil {
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
func (s *ConnectRPCServer) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
