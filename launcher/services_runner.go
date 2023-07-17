package launcher

import (
	"context"

	"github.com/tkcrm/micro/service"
)

type IServicesRunner interface {
	Register(svc *service.Service)
	Services() []*service.Service
}

type servicesRunner struct {
	services []*service.Service
	ctx      context.Context
}

func newServicesRunner(ctx context.Context) *servicesRunner {
	return &servicesRunner{
		services: make([]*service.Service, 0),
		ctx:      ctx,
	}
}

func (s *servicesRunner) Register(svc *service.Service) {
	if svc == nil {
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
