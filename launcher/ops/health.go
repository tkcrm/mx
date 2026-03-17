package ops

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/tkcrm/mx/launcher/types"
	"github.com/tkcrm/mx/logger"
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

// health implements service lifecycle
// and used as worker pool for HealthChecker.
type healthCheckerOpsService struct {
	log    logger.ExtendedLogger
	config HealthCheckerConfig

	resp *sync.Map
}

type HealthCheckerConfig struct {
	Enabled bool   `default:"false" usage:"allows to enable health checker" example:"true"`
	Path    string `default:"/healthy" validate:"required" usage:"allows to set custom healthy path" example:"/healthy"`
	Port    string `default:"10000" validate:"required" usage:"allows to set custom healthy port" example:"10000"`

	// LivenessPath is the HTTP path for the liveness probe (/livez).
	// Empty string disables the endpoint.
	LivenessPath string `default:"/livez" usage:"liveness probe path" example:"/livez"`

	// ReadinessPath is the HTTP path for the readiness probe (/readyz).
	// Empty string disables the endpoint.
	ReadinessPath string `default:"/readyz" usage:"readiness probe path" example:"/readyz"`

	servicesList []types.HealthChecker
	statesList   []types.StateProvider
}

func (s *HealthCheckerConfig) AddServicesList(list []types.HealthChecker) {
	s.servicesList = list
}

func (s *HealthCheckerConfig) AddStateList(list []types.StateProvider) {
	s.statesList = list
}

func newHealthCheckerOpsService(
	log logger.ExtendedLogger,
	config HealthCheckerConfig,
) *healthCheckerOpsService {
	if config.Path == "" {
		config.Path = "/healthy"
	}
	if config.LivenessPath == "" {
		config.LivenessPath = "/livez"
	}
	if config.ReadinessPath == "" {
		config.ReadinessPath = "/readyz"
	}
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

// Start implements service lifecycle for OPS health checker worker.
func (s *healthCheckerOpsService) Start(ctx context.Context) error {
	wg := new(sync.WaitGroup)

	wg.Add(len(s.config.servicesList))
	for i := range len(s.config.servicesList) {
		if s.config.servicesList[i] == nil {
			wg.Done()
			continue
		}

		s.resp.Store(s.config.servicesList[i].Name(), 0)

		go func(checker types.HealthChecker) {
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
	if s.config.LivenessPath != "" {
		mux.HandleFunc(s.config.LivenessPath, s.serveLiveness)
	}
	if s.config.ReadinessPath != "" {
		mux.HandleFunc(s.config.ReadinessPath, s.serveReadiness)
	}
}

// serveLiveness handles the /livez liveness probe.
// Returns 200 if no service is in Failed state, 503 otherwise.
// Liveness does not require HealthChecker — it reads ServiceState directly.
func (s *healthCheckerOpsService) serveLiveness(w http.ResponseWriter, _ *http.Request) {
	services := make(map[string]string, len(s.config.statesList))
	hasFailed := false

	for _, sp := range s.config.statesList {
		state := sp.State()
		services[sp.Name()] = state.String()
		if state == types.ServiceStateFailed {
			hasFailed = true
		}
	}

	status := "ok"
	resCode := http.StatusOK
	if hasFailed {
		status = "failed"
		resCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resCode)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":   status,
		"services": services,
	}); err != nil {
		s.log.Errorf("livez: could not write response: %s", err)
	}
}

type readinessServiceEntry struct {
	State  string `json:"state"`
	Health string `json:"health"`
}

// serveReadiness handles the /readyz readiness probe.
// Combines ServiceState with HealthChecker poll results.
// Returns 200 only when all services are Running and all health checks pass.
// Returns 424 if any service is Starting/Idle or a health check is still starting.
// Returns 503 if any service is Failed or a health check returned an error.
func (s *healthCheckerOpsService) serveReadiness(w http.ResponseWriter, _ *http.Request) {
	entries := make(map[string]*readinessServiceEntry, len(s.config.statesList))
	var existsErr, existsStarting bool

	// populate state for every registered service
	for _, sp := range s.config.statesList {
		state := sp.State()
		entry := &readinessServiceEntry{
			State:  state.String(),
			Health: "n/a",
		}
		entries[sp.Name()] = entry

		switch state {
		case types.ServiceStateFailed:
			existsErr = true
		case types.ServiceStateStarting, types.ServiceStateIdle:
			existsStarting = true
		}
	}

	// overlay HealthChecker poll results
	s.resp.Range(func(key, val any) bool {
		name, ok := key.(string)
		if !ok {
			return true
		}
		code, ok := val.(HealthCheckCode)
		if !ok {
			return true
		}

		entry, exists := entries[name]
		if !exists {
			entry = &readinessServiceEntry{State: "unknown"}
			entries[name] = entry
		}

		switch code {
		case HealthCheckCodeOk:
			entry.Health = "ok"
		case HealthCheckCodeError:
			entry.Health = "error"
			existsErr = true
		case HealthCheckCodeServiceStarting:
			entry.Health = "starting"
			existsStarting = true
		}
		return true
	})

	status := "ok"
	resCode := http.StatusOK
	if existsStarting {
		status = "starting"
		resCode = http.StatusFailedDependency
	}
	if existsErr {
		status = "unavailable"
		resCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resCode)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":   status,
		"services": entries,
	}); err != nil {
		s.log.Errorf("readyz: could not write response: %s", err)
	}
}

var _ opsService = (*healthCheckerOpsService)(nil)
