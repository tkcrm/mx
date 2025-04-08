// It's okay to expose pprof from this binary since the port it is exposed on
// is not accessible from the outside of Kubernetes cluster (only inside of it).
//
// #nosec G108 (CWE-200): Profiling endpoint is automatically exposed on /debug/pprof

package ops

import (
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

// New return list with ops services.
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
		names    []string
		httpOpts []http_transport.Option
		srv      *http.ServeMux
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
		muxServers[svc.getPort()].httpOpts = append(muxServers[svc.getPort()].httpOpts, svc.getHTTPOptions()...)

		svc.initService(muxServers[svc.getPort()].srv)
	}

	// append http servers
	for port, item := range muxServers {
		opts := []http_transport.Option{
			s.config.getHTTPOptionForPort(port),
			http_transport.WithLogger(s.logger),
			http_transport.WithHandler(item.srv),
			http_transport.WithName("ops-server-" + strings.Join(item.names, "-")),
			http_transport.WithWriteTimeout(60),
		}

		if len(item.httpOpts) > 0 {
			opts = append(opts, item.httpOpts...)
		}

		s.services = append(s.services, http_transport.NewServer(opts...))
	}

	return s.services
}
