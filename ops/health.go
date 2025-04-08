package ops

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"github.com/tkcrm/mx/transport/http_transport"
)

type HealthCheckCode int

const (
	HealthCheckCodeOk              HealthCheckCode = 0
	HealthCheckCodeError           HealthCheckCode = 1
	HealthCheckCodeServiceStarting HealthCheckCode = 2
)

var (
	ErrHealthCheckError           = errors.New("health check error")
	ErrHealthCheckServiceStarting = errors.New("service is starting")
)

// health implements service.Service
// and used as worker pool for HealthChecker.
type healthCheckerOpsService struct {
	log    logger.ExtendedLogger
	config HealthCheckerConfig

	resp *sync.Map
}

type HealthCheckerConfig struct {
	Enabled      bool   `default:"false" usage:"allows to enable health checker" example:"true"`
	Path         string `default:"/healthy" validate:"required" usage:"allows to set custom healthy path" example:"/healthy"`
	Port         string `default:"10000" validate:"required" usage:"allows to set custom healthy port" example:"10000"`
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
func (s *healthCheckerOpsService) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	var existsErr, existsProcessing bool
	out := make(map[string]interface{})
	s.resp.Range(func(key, val any) bool {
		if name, ok := key.(string); ok {
			out[name] = val
		}

		if code, ok := val.(HealthCheckCode); ok {
			switch code {
			case HealthCheckCodeError:
				existsErr = true
			case HealthCheckCodeServiceStarting:
				existsProcessing = true
			}
		}

		return true
	})

	w.Header().Add("Content-Type", "application/json")

	resCode := http.StatusOK
	if existsProcessing {
		resCode = http.StatusFailedDependency
	}

	if existsErr {
		resCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(resCode)

	if err := json.NewEncoder(w).Encode(out); err != nil {
		s.log.Errorf("could not write response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// implementation of service.IService for OPS worker.
func (s *healthCheckerOpsService) Start(ctx context.Context) error {
	wg := new(sync.WaitGroup)

	wg.Add(len(s.config.servicesList))
	for i := range len(s.config.servicesList) {
		if s.config.servicesList[i] == nil {
			wg.Done()
			continue
		}

		s.resp.Store(s.config.servicesList[i].Name(), 0)

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
					if err := checker.Healthy(ctx); err != nil { //nolint:nestif
						if errors.Is(err, ErrHealthCheckServiceStarting) {
							s.resp.Store(name, HealthCheckCodeServiceStarting)
						} else {
							s.resp.Store(name, HealthCheckCodeError)
						}
						s.log.Warnf("health check service %s failed with error: %s", name, err)
					} else {
						currentValue, existsValue := s.resp.Load(name)
						s.resp.Store(name, HealthCheckCodeOk)
						if existsValue {
							if code, ok := currentValue.(HealthCheckCode); ok && code != HealthCheckCodeOk {
								s.log.Infof("health check service %s fixed", name)
							}
						}
					}

					ticker.Reset(delay)
				}
			}
		}(s.config.servicesList[i])
	}

	<-ctx.Done()

	wg.Wait()

	return nil
}

func (s *healthCheckerOpsService) Stop(_ context.Context) error { return nil }

func (s healthCheckerOpsService) getEnabled() bool { return s.config.Enabled }

func (s healthCheckerOpsService) getPort() string { return s.config.Port }

func (s healthCheckerOpsService) getHTTPOptions() []http_transport.Option {
	res := make([]http_transport.Option, 0)
	return res
}

func (s *healthCheckerOpsService) initService(mux *http.ServeMux) {
	mux.Handle(s.config.Path, s)
}

var _ opsService = (*healthCheckerOpsService)(nil)
