package ops

import "github.com/tkcrm/mx/transport/http_transport"

// Config provides configuration for ops server.
type Config struct {
	Enabled bool   `default:"false" usage:"allows to enable ops server"`
	Addr    string `default:":10000" usage:"allows to set set ops address:port"`
	Network string `default:"tcp" usage:"allows to set ops listen network: tcp/udp"`
	NoTrace bool   `default:"true" usage:"allows to disable tracing"`

	MetricsPath string `default:"/metrics" usage:"allows to set custom metrics path"`
	HealthyPath string `default:"/healthy" usage:"allows to set custom healthy path"`
	ProfilePath string `default:"/debug/pprof" usage:"allows to set custom profiler path"`
}

func (o *Config) httpOption() http_transport.Option {
	return http_transport.WithConfig(http_transport.Config{
		Enabled: o.Enabled,
		Address: o.Addr,
		Network: o.Network,
		NoTrace: o.NoTrace,
	})
}
