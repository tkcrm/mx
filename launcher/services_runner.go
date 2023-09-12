package launcher

import (
	"context"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

type RunnerServicesSequence int

const (
	RunnerServicesSequenceNone = iota
	RunnerServicesSequenceFifo
	RunnerServicesSequenceLifo
)

type IServicesRunner interface {
	// Register servicse
	Register(services ...*service.Service)
	// Services return all registered services
	Services() []*service.Service
}

type servicesRunner struct {
	logger   logger.Logger
	services []*service.Service
	ctx      context.Context
}

func newServicesRunner(ctx context.Context, logger logger.Logger) *servicesRunner {
	return &servicesRunner{
		logger:   logger,
		services: make([]*service.Service, 0),
		ctx:      ctx,
	}
}

func (s *servicesRunner) Register(services ...*service.Service) {
	for _, svc := range services {
		s.registerService(svc)
	}
}

func (s *servicesRunner) registerService(svc *service.Service) {
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

func (s *servicesRunner) Services() []*service.Service {
	return s.services
}

// hcServices return services that implements HealthChecker interface
func (s *servicesRunner) hcServices() []service.HealthChecker {
	services := []service.HealthChecker{}
	for _, svc := range s.services {
		if svc.Options().HealthChecker != nil {
			services = append(services, svc.Options().HealthChecker)
		}
	}
	return services
}
