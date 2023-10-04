package ops

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type metricsOpsService struct {
	config MetricsConfig
}

func newMetricsOpsService(cfg MetricsConfig) *metricsOpsService {
	return &metricsOpsService{cfg}
}

type MetricsConfig struct {
	Enabled bool   `default:"true" usage:"allows to enable metrics"`
	Path    string `default:"/metrics" usage:"allows to set custom metrics path"`
	Port    string `default:"10000" usage:"allows to set custom metrics port"`
}

func (s metricsOpsService) Name() string { return "metrics" }

func (s metricsOpsService) getEnabled() bool { return s.config.Enabled }

func (s metricsOpsService) getPort() string { return s.config.Port }

func (s metricsOpsService) initService(mux *http.ServeMux) {
	mux.Handle(s.config.Path, promhttp.Handler())
}

var _ opsService = (*metricsOpsService)(nil)
