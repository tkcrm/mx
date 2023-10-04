package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

// health implements service.Service
// and used as worker pool for HealthChecker.
type healthCheckerOpsService struct {
	log    logger.ExtendedLogger
	config HealthCheckerConfig

	resp *sync.Map
}

type HealthCheckerConfig struct {
	Enabled      bool   `default:"true" usage:"allows to enable health checker"`
	Path         string `default:"/healthy" usage:"allows to set custom healthy path"`
	Port         string `default:"10000" usage:"allows to set custom healthy port"`
	servicesList []service.HealthChecker
}

func (s *HealthCheckerConfig) AddServicesList(list []service.HealthChecker) {
	s.servicesList = list
}

func newHealthCheckerOpsService(
	log logger.ExtendedLogger,
	config HealthCheckerConfig,
) *healthCheckerOpsService {
	return &healthCheckerOpsService{
		log:    log,
		config: config,
		resp:   new(sync.Map),
	}
}

// Name returns name of http server.
func (s healthCheckerOpsService) Name() string { return "ops-health-checker" }

// ServeHTTP implementation of http.Handler for OPS worker.
func (o *healthCheckerOpsService) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	out := make(map[string]interface{})
	o.resp.Range(func(key, val any) bool {
		if name, ok := key.(string); ok {
			out[name] = val
		}

		return true
	})

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(out); err != nil {
		o.log.Errorf("could not write response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// implementation of service.IService for OPS worker.
func (o *healthCheckerOpsService) Start(ctx context.Context) error {
	wg := new(sync.WaitGroup)

	wg.Add(len(o.config.servicesList))
	for i := 0; i < len(o.config.servicesList); i++ {
		if o.config.servicesList[i] == nil {
			wg.Done()
			continue
		}

		o.resp.Store(o.config.servicesList[i].Name(), 0)

		// run health checker for each service
		go func(checker service.HealthChecker) {
			defer wg.Done()

			name := checker.Name()
			delay := checker.Interval()

			ticker := time.NewTimer(delay)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := checker.Healthy(ctx); err != nil {
						o.resp.Store(name, 1)
						o.log.Warnf("check service %s failed with error: %s", name, err)
					} else {
						o.resp.Store(name, 0)
					}

					ticker.Reset(delay)
				}
			}
		}(o.config.servicesList[i])
	}

	<-ctx.Done()

	wg.Wait()

	return nil
}

func (o *healthCheckerOpsService) Stop(ctx context.Context) error { return nil }

func (s healthCheckerOpsService) getEnabled() bool { return s.config.Enabled }

func (s healthCheckerOpsService) getPort() string { return s.config.Port }

func (s *healthCheckerOpsService) initService(mux *http.ServeMux) {
	mux.Handle(s.config.Path, s)
}

var _ opsService = (*healthCheckerOpsService)(nil)
