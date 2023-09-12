package http_transport

import (
	"net/http"

	"github.com/tkcrm/mx/logger"
)

// Option allows customizing http component settings.
type Option func(*httpServer)

// WithConfig allows set custom http settings.
func WithConfig(v Config) Option {
	return func(s *httpServer) { s.Config = v }
}

// WithName allows set custom http name value.
func WithName(v string) Option {
	return func(s *httpServer) { s.name = v }
}

// WithLogger allows set custom logger value.
func WithLogger(v logger.ExtendedLogger) Option {
	return func(s *httpServer) { s.logger = v }
}

// WithHandler allows set custom http.Handler value.
func WithHandler(v http.Handler) Option {
	return func(s *httpServer) { s.handle = v }
}
