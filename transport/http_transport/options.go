package http_transport

import (
	"net/http"

	"github.com/tkcrm/mx/logger"
)

// Option allows customizing http component settings.
type Option func(*HttpServer)

// WithConfig allows set custom http settings.
func WithConfig(v Config) Option {
	return func(s *HttpServer) { s.Config = v }
}

// WithName allows set custom http name value.
func WithName(v string) Option {
	return func(s *HttpServer) { s.name = v }
}

// WithLogger allows set custom logger value.
func WithLogger(v logger.ExtendedLogger) Option {
	return func(s *HttpServer) { s.logger = v }
}

// WithHandler allows set custom http.Handler value.
func WithHandler(v http.Handler) Option {
	return func(s *HttpServer) { s.handle = v }
}

// WithReadTimeout allows set custom read timeout value.
func WithReadTimeout(v int) Option {
	return func(s *HttpServer) { s.ReadTimeout = v }
}

// WithWriteTimeout allows set custom write timeout value.
func WithWriteTimeout(v int) Option {
	return func(s *HttpServer) { s.WriteTimeout = v }
}

// WithIdleTimeout allows set custom idle timeout value.
func WithIdleTimeout(v int) Option {
	return func(s *HttpServer) { s.IdleTimeout = v }
}

// WithReadHeaderTimeout allows set custom read header timeout value.
func WithReadHeaderTimeout(v int) Option {
	return func(s *HttpServer) { s.ReadHeaderTimeout = v }
}
