package ops

import (
	"github.com/tkcrm/mx/transport/http_transport"
)

// Config provides configuration for ops server.
type Config struct {
	Enabled bool   `default:"false" usage:"allows to enable ops server" example:"false"`
	Network string `default:"tcp" required:"true" validate:"oneof=tcp udp" usage:"allows to set ops listen network: tcp/udp" example:"tcp"`

	// Tracing
	TracingEnabled bool `yaml:"tracing_enabled" default:"false" usage:"allows to enable tracing" example:"false"`

	// Metrics
	Metrics MetricsConfig

	// Health checker
	Healthy HealthCheckerConfig

	// Profiler
	Profiler ProfilerConfig
}

func (c *Config) getHttpOptionForPort(port string) http_transport.Option {
	return http_transport.WithConfig(http_transport.Config{
		Enabled: c.Enabled,
		Address: ":" + port,
		Network: c.Network,
		NoTrace: !c.TracingEnabled,
	})
}
