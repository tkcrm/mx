package grpc_client

import (
	"context"

	"google.golang.org/grpc"
)

// Config provides configuration for grpc cleint.
type Config struct {
	Name     string `default:"grpc-client" validate:"required" example:"backend-grpc-client"`
	Addr     string `validate:"required" usage:"grpc server address" example:"localhost:9000"`
	UseTls   bool   `yaml:"use_tls" default:"false" example:"false"`
	Insecure bool   `default:"false" example:"false"`
	ctx      context.Context
	grpsOpts []grpc.DialOption
}

type Option func(s *Config)

func WithGrpcOpt(opts ...grpc.DialOption) Option {
	return func(s *Config) {
		s.grpsOpts = append(s.grpsOpts, opts...)
	}
}

func WithName(v string) Option {
	return func(s *Config) {
		if v != "" {
			s.Name = v
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
