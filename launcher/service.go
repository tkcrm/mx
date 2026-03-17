package launcher

import (
	"context"
	"time"
)

// Service wraps a lifecycle-managed unit with Start/Stop functions and hooks.
type Service struct {
	opts ServiceOptions

	isStarted bool
	isStopped bool
}

// NewService creates a new Service.
func NewService(opts ...ServiceOption) *Service {
	return &Service{
		opts: newServiceOptions(opts...),
	}
}

func (s Service) Name() string { return s.opts.Name }

func (s *Service) Options() *ServiceOptions { return &s.opts }

func (s Service) String() string { return "mx" }

func (s *Service) Start() error {
	if s.opts.StartFn == nil {
		return nil
	}

	if !s.opts.Enabled {
		s.opts.Logger.Infof("service [%s] was skipped because it is disabled", s.Name())
		return nil
	}

	if s.isStarted {
		return nil
	}
	s.isStarted = true
	s.isStopped = false

	s.opts.Logger.Infof("starting service [%s]", s.Name())

	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	errChan := make(chan error, 1)
	doneChan := make(chan struct{}, 1)
	go func() {
		if err := s.opts.StartFn(s.opts.Context); err != nil {
			errChan <- err
			return
		}
		doneChan <- struct{}{}
	}()

	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	select {
	case err := <-errChan:
		return err
	case <-s.opts.Context.Done():
	}

	// grace stop Start func
	select {
	case <-time.After(s.opts.ShutdownTimeout):
		s.opts.Logger.Infof("service [%s] was stopped by timeout", s.Name())
	case <-doneChan:
	}

	for _, fn := range s.opts.AfterStartFinished {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) Stop() error {
	if s.opts.StopFn == nil {
		return nil
	}

	if s.isStopped || !s.isStarted {
		return nil
	}
	s.isStarted = false
	s.isStopped = true

	var stopErr error

	for _, fn := range s.opts.BeforeStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opts.ShutdownTimeout)
	defer cancel()

	s.opts.Logger.Infoln("stopping service", s.Name())

	errChan := make(chan error, 1)
	doneChan := make(chan struct{}, 1)
	go func() {
		if err := s.opts.StopFn(ctx); err != nil {
			errChan <- err
			return
		}
		doneChan <- struct{}{}
	}()

	select {
	case <-doneChan:
		s.opts.Logger.Infof("service [%s] was stopped", s.Name())
	case err := <-errChan:
		return err
	case <-ctx.Done():
		s.opts.Logger.Infof("failed to stop service [%s]. stop by timeout", s.Name())
	}

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	return stopErr
}
