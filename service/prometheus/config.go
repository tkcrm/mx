package prometheus

import "fmt"

type Config struct {
	// Port - default 10001
	Port string `json:"PROMETHEUS_PORT" default:"10001"`
	// Endpoint - default /metrics
	Endpoint string `json:"PROMETHEUS_ENDPOINT" default:"/metrics"`
	// Enabled - default true
	Enabled bool `json:"PROMETHEUS_ENABLED" default:"true"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Port == "" {
		return fmt.Errorf("empty PROMETHEUS_PORT")
	}

	if c.Endpoint == "" {
		return fmt.Errorf("empty PROMETHEUS_ENDPOINT")
	}

	return nil
}
