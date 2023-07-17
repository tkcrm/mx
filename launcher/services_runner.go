package launcher

import (
	"context"

	"github.com/tkcrm/micro/logger"
	"github.com/tkcrm/micro/service"
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

	// skip service if it disabled
	if !svc.Options().Enabled {
		s.logger.Infof("service [%s] was skipped because it is disabled", svc.Name())
		return
	}

	// set context
	if svc.Options().Context == nil {
		svc.Options().Context = s.ctx
	}

	s.services = append(s.services, svc)
}

func (s *servicesRunner) Services() []*service.Service {
	return s.services
}