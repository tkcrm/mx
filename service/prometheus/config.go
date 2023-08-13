package prometheus

import "fmt"

type Config struct {
	// Port - default 10001
	Port string `default:"10001"`
	// Endpoint - default /metrics
	Endpoint string `default:"/metrics"`
	// Enabled - default true
	Enabled bool `default:"true"`
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
