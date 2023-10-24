package http_transport

// Config provides configuration for http server.
type Config struct {
	Enabled bool   `env:"ENABLED" default:"false" usage:"allows to enable http server" example:"true"`
	Address string `env:"ADDRESS" default:":8080" validate:"required,hostname_port" usage:"HTTP server listen address" example:"localhost:9000"`
	Network string `env:"NETWORK" default:"tcp" validate:"required" usage:"HTTP server listen network: tpc/udp" example:"tcp"`
	NoTrace bool   `env:"NO_TRACE" default:"false" usage:"allows to disable tracing for HTTP server" example:"false"`
}
