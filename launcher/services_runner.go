package launcher

import (
	"context"

	"github.com/tkcrm/mx/launcher/types"
	"github.com/tkcrm/mx/logger"
)

type RunnerServicesSequence int

const (
	RunnerServicesSequenceNone = iota
	RunnerServicesSequenceFifo
	RunnerServicesSequenceLifo
)

type IServicesRunner interface {
	// Register services
	Register(services ...*Service)
	// Services return all registered services
	Services() []*Service
	// Get returns a registered service by name, or false if not found.
	Get(name string) (*Service, bool)
}

type servicesRunner struct {
	logger   logger.Logger
	services []*Service
	ctx      context.Context //nolint:containedctx
}

func newServicesRunner(ctx context.Context, logger logger.Logger) *servicesRunner {
	return &servicesRunner{
		logger:   logger,
		services: make([]*Service, 0),
		ctx:      ctx,
	}
}

func (s *servicesRunner) Register(services ...*Service) {
	for _, svc := range services {
		s.registerService(svc)
	}
}

func (s *servicesRunner) registerService(svc *Service) {
	if svc == nil {
		s.logger.Error("trying to register nil service")
		return
	}

	svcOpts := svc.Options()

	// set context
	if svcOpts.Context == nil {
		svcOpts.Context = s.ctx
	}

	// set logger
	svcOpts.Logger = s.logger

	// validate service options
	if err := svcOpts.Validate(); err != nil {
		s.logger.Errorf("service [%s] was skipped because it has validation error: %s", svc.Name(), err)
		return
	}

	// skip service if it disabled
	if !svcOpts.Enabled {
		s.logger.Infof("service [%s] was skipped because it is disabled", svc.Name())
		return
	}

	s.services = append(s.services, svc)
}

func (s *servicesRunner) Services() []*Service {
	return s.services
}

// Get returns a registered service by name, or false if not found.
func (s *servicesRunner) Get(name string) (*Service, bool) {
	for _, svc := range s.services {
		if svc.Name() == name {
			return svc, true
		}
	}
	return nil, false
}

// hcServices return services that implement the HealthChecker interface.
func (s *servicesRunner) hcServices() []types.HealthChecker {
	services := []types.HealthChecker{}
	for _, svc := range s.services {
		if svc.Options().HealthChecker != nil {
			services = append(services, svc.Options().HealthChecker)
		}
	}
	return services
}

// stateProviders returns all registered services as StateProvider.
func (s *servicesRunner) stateProviders() []types.StateProvider {
	providers := make([]types.StateProvider, len(s.services))
	for i, svc := range s.services {
		providers[i] = svc
	}
	return providers
}
