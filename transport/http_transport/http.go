package http_transport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/tkcrm/mx/logger"
)

type HTTPServer struct {
	Config

	name   string
	handle http.Handler
	server *http.Server
	logger logger.ExtendedLogger
}

const defaultHTTPName = "http-server"

// NewServer creates http server.
func NewServer(opts ...Option) *HTTPServer {
	serve := &HTTPServer{
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
func (s HTTPServer) Name() string { return s.name }

// Enabled returns is service enabled.
func (s HTTPServer) Enabled() bool { return s.Config.Enabled }

// Start allows starting http server.
func (s *HTTPServer) Start(ctx context.Context) error {
	log := logger.WithExtended(
		s.logger,
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

	if s.ReadTimeout == 0 {
		s.ReadTimeout = 5
	}

	if s.WriteTimeout == 0 {
		s.WriteTimeout = 10
	}

	if s.IdleTimeout == 0 {
		s.IdleTimeout = 60
	}

	if s.ReadHeaderTimeout == 0 {
		s.ReadHeaderTimeout = 10
	}

	s.server = &http.Server{
		// to prevent default std logger output
		Handler:  handler,
		ErrorLog: log.Std(),

		// G112: Potential Slowloris Attack because ReadHeaderTimeout is not configured in the http.Server
		ReadHeaderTimeout: time.Duration(s.ReadHeaderTimeout) * time.Second,

		ReadTimeout:  time.Duration(s.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.IdleTimeout) * time.Second,
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
	}

	return nil
}

// Stop allows stop http server.
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	return nil
}
