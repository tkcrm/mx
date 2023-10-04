package ops

import (
	"github.com/tkcrm/mx/transport/http_transport"
)

// Config provides configuration for ops server.
type Config struct {
	Enabled bool   `default:"false" usage:"allows to enable ops server"`
	Network string `default:"tcp" usage:"allows to set ops listen network: tcp/udp"`

	// Tracing
	TracingEnabled bool `default:"false" usage:"allows to enable tracing"`

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
