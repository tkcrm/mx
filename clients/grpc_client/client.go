package grpc_client

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func New[T any](
	config Config,
	logger logger,
	fn func(cc grpc.ClientConnInterface) T,
	opts ...Option,
) (T, error) {
	var nilIface T
	if fn == nil {
		return nilIface, fmt.Errorf("empty init func")
	}

	for _, o := range opts {
		o(&config)
	}

	if config.Name == "" {
		return nilIface, fmt.Errorf("empty name for grpc client")
	}

	if config.Addr == "" {
		return nilIface, fmt.Errorf("empty address")
	}

	if config.UseTls {
		config.grpsOpts = append(config.grpsOpts,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: false})),
		)
	}

	if config.Insecure {
		config.grpsOpts = append(config.grpsOpts,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	var conn *grpc.ClientConn
	if config.ctx == nil {
		c, err := grpc.Dial(config.Addr, config.grpsOpts...)
		if err != nil {
			return nilIface, fmt.Errorf(
				"grpc dial with [%s] server by client [%s] error: %w",
				config.Addr, config.Name, err,
			)
		}
		conn = c
	} else {
		c, err := grpc.DialContext(config.ctx, config.Addr, config.grpsOpts...)
		if err != nil {
			return nilIface, fmt.Errorf(
				"grpc dial with [%s] server by client [%s] error: %w",
				config.Addr, config.Name, err,
			)
		}
		conn = c
	}

	// if conn.GetState() != connectivity.Connecting {
	// 	return nilIface, fmt.Errorf(
	// 		"failed to ping [%s] grpc server by client [%s]",
	// 		config.Addr, config.Name,
	// 	)
	// }

	if logger != nil {
		logger.Infof("register grpc client: %s", config.Name)
	}

	return fn(conn), nil
}
