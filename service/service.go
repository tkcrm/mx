package service

import (
	"context"
	"os"
	"os/signal"
	"time"

	signalutil "github.com/tkcrm/micro/util/signal"
)

type Service struct {
	opts Options
}

func New(opts ...Option) *Service {
	return &Service{
		opts: newOptions(opts...),
	}
}

func (s *Service) Name() string { return s.opts.Name }

func (s *Service) Options() *Options { return &s.opts }

func (s *Service) String() string { return "micro" }

func (s *Service) Start() error {
	if s.opts.Start == nil {
		return nil
	}

	if !s.opts.Enabled {
		s.opts.logger.Infoln("service", s.Name(), "was skipped because it is disabled")
		return nil
	}

	s.opts.logger.Infoln("starting service", s.Name())

	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	var errChan = make(chan error, 1)
	go func() {
		if err := s.opts.Start(s.opts.Context); err != nil {
			errChan <- err
		}
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

	return nil
}

func (s *Service) Stop() error {
	if s.opts.Stop == nil {
		return nil
	}

	var stopErr error

	for _, fn := range s.opts.BeforeStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	ctx, cancel := context.WithTimeout(s.opts.Context, time.Second*10)
	defer cancel()

	//s.opts.logger.Infoln("stopping service", s.Name())

	var errChan = make(chan error, 1)
	var stopChan = make(chan struct{}, 1)
	go func() {
		if err := s.opts.Stop(ctx); err != nil {
			errChan <- err
		}
		stopChan <- struct{}{}
	}()

	select {
	// success stop
	case <-stopChan:
	// stop with error
	case err := <-errChan:
		return err
	// stop by context
	case <-ctx.Done():
	}

	s.opts.logger.Infoln("service", s.Name(), "stopped")

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	return stopErr
}
