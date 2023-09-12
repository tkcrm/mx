// It's okay to expose pprof from this binary since the port it is exposed on
// is not accessible from the outside of Kubernetes cluster (only inside of it).
//
// #nosec G108 (CWE-200): Profiling endpoint is automatically exposed on /debug/pprof

package ops

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"github.com/tkcrm/mx/transport/http_transport"
)

const (
	opsServiceName = "ops-server"
)

type ops struct {
	logger     logger.ExtendedLogger
	config     Config
	httpServer service.IService
	hcService  *healthChecker
}

// New creates new OPS server and OPS HealthChecker's worker.
func New(log logger.ExtendedLogger, cfg Config, hcServicesList ...service.HealthChecker) []service.IService {
	hcService := newHealthChecker(log, hcServicesList...)
	opsSvc := &ops{
		logger:    log,
		config:    cfg,
		hcService: hcService,
	}

	return []service.IService{
		hcService, opsSvc,
	}
}

// Name returns name of http server.
func (s ops) Name() string { return opsServiceName }

// Enabled returns is service enabled.
func (s ops) Enabled() bool { return s.config.Enabled }

// Start ops server
func (s *ops) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// prepare HealthChecker's worker and handler
	mux.Handle(s.config.HealthyPath, s.hcService)

	// Expose the registered pprof via HTTP.
	mux.HandleFunc(s.config.ProfilePath+"/", pprof.Index)
	mux.HandleFunc(s.config.ProfilePath+"/cmdline", pprof.Cmdline)
	mux.HandleFunc(s.config.ProfilePath+"/profile", pprof.Profile)
	mux.HandleFunc(s.config.ProfilePath+"/symbol", pprof.Symbol)
	mux.HandleFunc(s.config.ProfilePath+"/trace", pprof.Trace)

	// metrics
	mux.Handle(s.config.MetricsPath, promhttp.Handler())

	srv := http_transport.NewServer(
		s.config.httpOption(),
		http_transport.WithLogger(s.logger),
		http_transport.WithHandler(mux),
		http_transport.WithName(opsServiceName),
	)

	s.httpServer = srv

	return s.httpServer.Start(ctx)
}

// Stop ops server
func (s *ops) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		if err := s.httpServer.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}
