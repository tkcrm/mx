package service

import (
	"context"
	"os"
	"os/signal"
	"time"

	signalutil "github.com/tkcrm/mx/util/signal"
)

type IService interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Service struct {
	opts Options

	isStarted bool
	isStopped bool
}

func New(opts ...Option) *Service {
	return &Service{
		opts: newOptions(opts...),
	}
}

func (s Service) Name() string { return s.opts.Name }

func (s *Service) Options() *Options { return &s.opts }

func (s Service) String() string { return "mx" }

func (s *Service) Start() error {
	if s.opts.StartFn == nil {
		return nil
	}

	if !s.opts.Enabled {
		s.opts.Logger.Infof("service [%s] was skipped because it is disabled", s.Name())
		return nil
	}

	// skip if service already started
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

	ch := make(chan os.Signal, 1)
	if s.opts.Signal {
		signal.Notify(ch, signalutil.Shutdown()...)
	}

	select {
	// wait on service error
	case err := <-errChan:
		return err
	// wait on kill signal
	case <-ch:
	// wait on context cancel
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

	// skip if service already stopped or not started
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
	// success stop
	case <-doneChan:
		s.opts.Logger.Infof("service [%s] was stopped", s.Name())
	// stop with error
	case err := <-errChan:
		return err
	// stop by context
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
