package connectrpc_client

import (
	"context"

	"connectrpc.com/connect"
)

// Config provides configuration for grpc cleint.
type Config struct {
	Name           string `validate:"required" default:"connectrpc-client" example:"backend-connectrpc-client"`
	Addr           string `usage:"connectrpc server address" example:"localhost:9000"`
	ctx            context.Context
	httpClient     connect.HTTPClient
	connectrpcOpts []connect.ClientOption
}

type Option func(s *Config)

func WithConnectrpcOpts(opts ...connect.ClientOption) Option {
	return func(s *Config) {
		s.connectrpcOpts = append(s.connectrpcOpts, opts...)
	}
}

func WithName(v string) Option {
	return func(s *Config) {
		if v != "" {
			s.Name = v
		}
	}
}

func WithHttpClient(c connect.HTTPClient) Option {
	return func(s *Config) {
		if c != nil {
			s.httpClient = c
		}
	}
}

func WithContext(v context.Context) Option {
	return func(s *Config) {
		if v != nil {
			s.ctx = v
		}
	}
}
