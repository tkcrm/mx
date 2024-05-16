package connectrpc_client

import (
	"fmt"
	"net/http"

	"connectrpc.com/connect"
)

func New[T any](
	config Config,
	logger logger,
	fn func(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) T,
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
		return nilIface, fmt.Errorf("empty name for connectrpc client")
	}

	if config.Addr == "" {
		return nilIface, fmt.Errorf("empty address")
	}

	if config.httpClient == nil {
		config.httpClient = http.DefaultClient
	}

	if logger != nil {
		logger.Infof("register connectrpc client: %s", config.Name)
	}

	return fn(config.httpClient, config.Addr, config.connectrpcOpts...), nil
}
