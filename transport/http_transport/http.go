package http_transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/tkcrm/mx/logger"
)

type httpServer struct {
	Config

	name   string
	handle http.Handler
	server *http.Server
	logger logger.ExtendedLogger
}

const defaultHTTPName = "http-server"

// NewServer creates http server.
func NewServer(opts ...Option) *httpServer {
	serve := &httpServer{
		name:   defaultHTTPName,
		logger: logger.DefaultExtended(),

		Config: Config{Enabled: true},
	}

	for _, o := range opts {
		o(serve)
	}

	return serve
}

// Name returns name of http server.
func (s httpServer) Name() string { return s.name }

// Enabled returns is service enabled.
func (s httpServer) Enabled() bool { return s.Config.Enabled }

// Start allows starting http server.
func (s *httpServer) Start(ctx context.Context) error {
	log := s.logger.With(
		"name", s.name,
		"address", s.Address,
		"network", s.Network,
	)

	log.Infof("prepare listener %s on %s / %s",
		s.name, s.Address, s.Network,
	)

	lis, err := new(net.ListenConfig).Listen(ctx, s.Network, s.Address)
	if err != nil {
		return err
	}

	handler := s.handle
	if !s.NoTrace {
		handler = TracingMiddleware(s.handle)
	}

	s.server = &http.Server{
		// to prevent default std logger output
		Handler:  handler,
		ErrorLog: log.Std(),

		// G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server
		ReadHeaderTimeout: time.Second * 10,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
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

// Stop allows stop http server.
func (s *httpServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("could not stop server: %w", err)
	}

	return nil
}
