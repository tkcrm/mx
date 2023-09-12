package http_transport

// Config provides configuration for http server.
type Config struct {
	Enabled bool   `env:"ENABLED" default:"false" usage:"allows to enable http server"`
	Address string `env:"ADDRESS" default:":8080" usage:"HTTP server listen address"`
	Network string `env:"NETWORK" default:"tcp" usage:"HTTP server listen network: tpc/udp"`
	NoTrace bool   `env:"NO_TRACE" default:"false" usage:"allows to disable tracing for HTTP server"`
}
