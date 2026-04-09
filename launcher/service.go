package launcher

import (
	"context"
	"fmt"
	"time"

	"github.com/tkcrm/mx/launcher/types"
)

// ServiceState is an alias for types.ServiceState for convenience.
type ServiceState = types.ServiceState

// Re-export ServiceState constants so callers don't need to import launcher/types.
const (
	ServiceStateIdle     = types.ServiceStateIdle
	ServiceStateStarting = types.ServiceStateStarting
	ServiceStateRunning  = types.ServiceStateRunning
	ServiceStateStopping = types.ServiceStateStopping
	ServiceStateStopped  = types.ServiceStateStopped
	ServiceStateFailed   = types.ServiceStateFailed
)

// Service wraps a lifecycle-managed unit with Start/Stop functions and hooks.
type Service struct {
	opts    ServiceOptions
	state   ServiceState
	readyCh chan struct{}
}

// NewService creates a new Service.
func NewService(opts ...ServiceOption) *Service {
	return &Service{
		opts:    newServiceOptions(opts...),
		readyCh: make(chan struct{}),
	}
}

// Ready returns a channel that is closed when the service transitions to Running state.
func (s *Service) Ready() <-chan struct{} { return s.readyCh }

func (s Service) Name() string { return s.opts.Name }

// State returns the current lifecycle state of the service.
func (s *Service) State() ServiceState { return s.state }

func (s *Service) Options() *ServiceOptions { return &s.opts }

func (s Service) String() string { return "mx" }

func (s *Service) Start() error {
	defer func() {
		select {
		case <-s.readyCh:
		default:
			close(s.readyCh)
		}
	}()

	if s.opts.StartFn == nil {
		return nil
	}

	if !s.opts.Enabled {
		s.opts.Logger.Infof("service [%s] was skipped because it is disabled", s.Name())
		return nil
	}

	if s.state == ServiceStateStarting || s.state == ServiceStateRunning {
		return nil
	}
	s.state = ServiceStateStarting

	s.opts.Logger.Infof("starting service [%s]", s.Name())

	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			s.state = ServiceStateFailed
			return err
		}
	}

	err := s.runWithRestarts()

	for _, fn := range s.opts.AfterStartFinished {
		if fnErr := fn(); fnErr != nil {
			return fnErr
		}
	}

	return err
}

// runWithRestarts runs StartFn in a loop, restarting according to RestartPolicy.
// AfterStart hooks fire once before entering the loop.
func (s *Service) runWithRestarts() error {
	policy := s.opts.RestartPolicy

	// AfterStart hooks run once, immediately after the first goroutine is launched.
	afterStartDone := false

	for attempt := 0; ; attempt++ {
		errChan := make(chan error, 1)
		doneChan := make(chan struct{}, 1)
		go func() {
			if err := s.opts.StartFn(s.opts.Context); err != nil {
				errChan <- err
				return
			}
			doneChan <- struct{}{}
		}()

		s.state = ServiceStateRunning

		select {
		case <-s.readyCh:
		default:
			close(s.readyCh)
		}

		if !afterStartDone {
			afterStartDone = true
			for _, fn := range s.opts.AfterStart {
				if err := fn(); err != nil {
					s.state = ServiceStateFailed
					return err
				}
			}
		}

		// wait for exit: error, clean finish, startup timeout, or context cancel
		var exitErr error
		var cleanExit bool

		if s.opts.StartupTimeout > 0 {
			select {
			case err := <-errChan:
				exitErr = err
			case <-doneChan:
				cleanExit = true
			case <-time.After(s.opts.StartupTimeout):
				exitErr = fmt.Errorf("service [%s] startup timeout exceeded (%s)", s.Name(), s.opts.StartupTimeout)
			case <-s.opts.Context.Done():
				select {
				case <-time.After(s.opts.ShutdownTimeout):
					s.opts.Logger.Infof("service [%s] was stopped by timeout", s.Name())
				case <-doneChan:
				case <-errChan:
				}
				return nil
			}
		} else {
			select {
			case err := <-errChan:
				exitErr = err
			case <-doneChan:
				cleanExit = true
			case <-s.opts.Context.Done():
				select {
				case <-time.After(s.opts.ShutdownTimeout):
					s.opts.Logger.Infof("service [%s] was stopped by timeout", s.Name())
				case <-doneChan:
				case <-errChan:
				}
				return nil
			}
		}

		// determine whether to restart
		shouldRestart := false
		switch policy.Mode {
		case RestartOnFailure:
			shouldRestart = exitErr != nil
		case RestartAlways:
			shouldRestart = true
		}

		if !shouldRestart {
			if exitErr != nil {
				s.state = ServiceStateFailed
				return exitErr
			}
			_ = cleanExit
			return nil
		}

		delay, allowed := policy.nextDelay(attempt)
		if !allowed {
			s.state = ServiceStateFailed
			if exitErr != nil {
				return fmt.Errorf("service [%s] failed after %d restart attempt(s): %w", s.Name(), attempt+1, exitErr)
			}
			return nil
		}

		if exitErr != nil {
			s.opts.Logger.Warnf("service [%s] failed (attempt %d), restarting in %s: %s", s.Name(), attempt+1, delay, exitErr)
		} else {
			s.opts.Logger.Infof("service [%s] exited cleanly (attempt %d), restarting in %s", s.Name(), attempt+1, delay)
		}

		s.state = ServiceStateStarting
		select {
		case <-time.After(delay):
		case <-s.opts.Context.Done():
			return nil
		}
	}
}

func (s *Service) Stop() error {
	if s.opts.StopFn == nil {
		return nil
	}

	if s.state == ServiceStateStopped || s.state == ServiceStateStopping || s.state == ServiceStateIdle {
		return nil
	}
	s.state = ServiceStateStopping

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
		s.state = ServiceStateStopped
		s.opts.Logger.Infof("service [%s] was stopped", s.Name())
	case err := <-errChan:
		s.state = ServiceStateFailed
		return err
	case <-ctx.Done():
		s.state = ServiceStateStopped
		s.opts.Logger.Infof("failed to stop service [%s]. stop by timeout", s.Name())
	}

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	return stopErr
}
