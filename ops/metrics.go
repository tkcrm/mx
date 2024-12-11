package ops

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tkcrm/mx/transport/http_transport"
)

type metricsOpsService struct {
	config MetricsConfig
}

func newMetricsOpsService(cfg MetricsConfig) *metricsOpsService {
	return &metricsOpsService{cfg}
}

type MetricsConfig struct {
	Enabled   bool                           `default:"false" usage:"allows to enable metrics" example:"true"`
	Path      string                         `default:"/metrics" validate:"required" usage:"allows to set custom metrics path" example:"/metrics"`
	Port      string                         `default:"10000" validate:"required" usage:"allows to set custom metrics port" example:"10000"`
	BasicAuth http_transport.BasicAuthConfig `yaml:"basic_auth"`
}

func (s metricsOpsService) Name() string { return "metrics" }

func (s metricsOpsService) getEnabled() bool { return s.config.Enabled }

func (s metricsOpsService) getPort() string { return s.config.Port }

func (s metricsOpsService) getHttpOptions() []http_transport.Option {
	res := make([]http_transport.Option, 0)
	return res
}

func (s metricsOpsService) initService(mux *http.ServeMux) {
	mux.Handle(s.config.Path, http_transport.BasicAuthHandler(promhttp.Handler(), s.config.BasicAuth))
}

var _ opsService = (*metricsOpsService)(nil)
