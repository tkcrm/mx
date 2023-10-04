// It's okay to expose pprof from this binary since the port it is exposed on
// is not accessible from the outside of Kubernetes cluster (only inside of it).
//
// #nosec G108 (CWE-200): Profiling endpoint is automatically exposed on /debug/pprof

package ops

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"github.com/tkcrm/mx/transport/http_transport"
)

type ops struct {
	logger   logger.ExtendedLogger
	config   Config
	services []service.IService
}

// New return list with ops services
func New(log logger.ExtendedLogger, cfg Config) []service.IService {
	s := &ops{
		logger: log,
		config: cfg,
	}

	services := []opsService{}

	if s.config.Metrics.Enabled {
		services = append(services, newMetricsOpsService(s.config.Metrics))
	}

	if s.config.Healthy.Enabled {
		hcSvc := newHealthCheckerOpsService(s.logger, s.config.Healthy)
		s.services = append(s.services, hcSvc)
		services = append(services, hcSvc)
	}

	if s.config.Profiler.Enabled {
		services = append(services, newProfilerOpsService(s.config.Profiler))
	}

	if len(services) == 0 {
		return []service.IService{}
	}

	type muxServer struct {
		names []string
		srv   *http.ServeMux
	}
	muxServers := map[string]*muxServer{}

	// init servers for different ports
	for _, svc := range services {
		if _, ok := muxServers[svc.getPort()]; !ok {
			muxServers[svc.getPort()] = &muxServer{
				srv: http.NewServeMux(),
			}
		}

		muxServers[svc.getPort()].names = append(muxServers[svc.getPort()].names, svc.Name())
		svc.initService(muxServers[svc.getPort()].srv)
	}

	// append http servers
	for port, item := range muxServers {
		s.services = append(s.services, http_transport.NewServer(
			s.config.getHttpOptionForPort(port),
			http_transport.WithLogger(s.logger),
			http_transport.WithHandler(item.srv),
			http_transport.WithName(fmt.Sprintf("ops-server-%s", strings.Join(item.names, "-"))),
		))
	}

	return s.services
}
