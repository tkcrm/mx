package http_transport

// Config provides configuration for http server.
type Config struct {
	Enabled           bool   `default:"false" usage:"allows to enable http server" example:"true"`
	Address           string `default:":8080" validate:"required,hostname_port" usage:"HTTP server listen address" example:"localhost:9000"`
	Network           string `default:"tcp" validate:"required" usage:"HTTP server listen network: tpc/udp" example:"tcp"`
	NoTrace           bool   `yaml:"no_trace" default:"false" usage:"allows to disable tracing for HTTP server" example:"false"`
	ReadTimeout       int    `yaml:"read_timeout" default:"5" usage:"HTTP server read timeout in seconds" example:"5"`
	WriteTimeout      int    `yaml:"write_timeout" default:"10" usage:"HTTP server write timeout in seconds" example:"10"`
	IdleTimeout       int    `yaml:"idle_timeout" default:"60" usage:"HTTP server idle timeout in seconds" example:"60"`
	ReadHeaderTimeout int    `yaml:"read_header_timeout" default:"10" usage:"HTTP server read header timeout in seconds" example:"10"`
}
