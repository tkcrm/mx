package grpc_client

import "google.golang.org/grpc"

// Config provides configuration for grpc cleint.
type Config struct {
	Enabled  bool   `default:"true" usage:"allows to enable grpc client"`
	Name     string `default:"nameless-grpc-client"`
	Addr     string `validate:"hostname_port" usage:"grpc server address"`
	UseTls   bool
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
