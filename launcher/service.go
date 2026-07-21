package launcher

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/tkcrm/mx/mxtypes"
)

// ServiceState is an alias for types.ServiceState for convenience.
type ServiceState = mxtypes.ServiceState

// Re-export ServiceState constants so callers don't need to import launcher/types.
const (
	ServiceStateIdle     = mxtypes.ServiceStateIdle
	ServiceStateStarting = mxtypes.ServiceStateStarting
	ServiceStateRunning  = mxtypes.ServiceStateRunning
	ServiceStateStopping = mxtypes.ServiceStateStopping
	ServiceStateStopped  = mxtypes.ServiceStateStopped
	ServiceStateFailed   = mxtypes.ServiceStateFailed
)

// Service wraps a lifecycle-managed unit with Start/Stop functions and hooks.
type Service struct {
	opts ServiceOptions
	// state holds the current ServiceState as an int32 so it can be read
	// (State) from ops probe handlers concurrently with the service goroutine
	// updating it, without a data race.
	state   atomic.Int32
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

func (s *Service) Name() string { return s.opts.Name }

// State returns the current lifecycle state of the service.
func (s *Service) State() ServiceState { return ServiceState(s.state.Load()) }

// setState atomically updates the service's lifecycle state.
func (s *Service) setState(v ServiceState) { s.state.Store(int32(v)) }

func (s *Service) Options() *ServiceOptions { return &s.opts }

func (s *Service) String() string { return "mx" }

func (s *Service) Start() error {
	// Ensure the readiness channel is always closed on return, so the launcher's
	// startup-priority barrier never blocks on a service that failed or exited
	// before signalling ready.
	defer s.closeReady()

	if s.opts.StartFn == nil {
		return nil
	}

	if !s.opts.Enabled {
		s.opts.Logger.Infof("service [%s] was skipped because it is disabled", s.Name())
		return nil
	}

	if s.State() == ServiceStateStarting || s.State() == ServiceStateRunning {
		return nil
	}
	s.setState(ServiceStateStarting)

	s.opts.Logger.Infof("starting service [%s]", s.Name())

	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			s.setState(ServiceStateFailed)
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
//
// Each start goes through two phases:
//   - startup: wait for the service to report readiness, bounded by
//     StartupTimeout. Services without a readiness reporter are ready at once.
//   - steady state: run until the service exits or the context is cancelled;
//     StartupTimeout does not apply here, so a healthy long-running service is
//     never killed by it.
//
// Readiness is gated only on the first attempt; restarts go straight back to
// Running. AfterStart hooks fire once, right after the service first becomes
// ready.
func (s *Service) runWithRestarts() error {
	policy := s.opts.RestartPolicy
	gatedReady := false

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

		var exitErr error
		exitedDuringStartup := false

		if !gatedReady {
			gatedReady = true

			// Startup phase: wait for the service to report readiness, bounded by
			// StartupTimeout. A service that fails, exits, or times out before
			// signalling ready never reaches the Running state.
			if s.opts.Readiness != nil {
				var timeout <-chan time.Time
				if s.opts.StartupTimeout > 0 {
					timer := time.NewTimer(s.opts.StartupTimeout)
					defer timer.Stop()
					timeout = timer.C
				}

				select {
				case <-s.opts.Readiness():
					// service reported ready
				case err := <-errChan:
					exitErr, exitedDuringStartup = err, true
				case <-doneChan:
					exitedDuringStartup = true
				case <-timeout:
					exitErr = fmt.Errorf("service [%s] startup timeout exceeded (%s)", s.Name(), s.opts.StartupTimeout)
					exitedDuringStartup = true
				case <-s.opts.Context.Done():
					return s.awaitShutdown(errChan, doneChan)
				}
			}

			switch {
			case !exitedDuringStartup:
				// Became ready.
				s.setState(ServiceStateRunning)
				s.closeReady()
				if err := s.runAfterStart(); err != nil {
					return err
				}
			case exitErr == nil:
				// Exited cleanly before signalling ready — it ran to completion,
				// so reflect that as Running (consistent with services that do
				// not report readiness) rather than leaving it stuck in Starting.
				s.setState(ServiceStateRunning)
			}
		} else {
			// Restart: the service already proved readiness once — back to Running.
			s.setState(ServiceStateRunning)
		}

		// Steady-state phase: wait for the service to exit or be cancelled.
		if !exitedDuringStartup {
			select {
			case err := <-errChan:
				exitErr = err
			case <-doneChan:
				// clean exit
			case <-s.opts.Context.Done():
				return s.awaitShutdown(errChan, doneChan)
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
				s.setState(ServiceStateFailed)
				return exitErr
			}
			return nil
		}

		delay, allowed := policy.nextDelay(attempt)
		if !allowed {
			s.setState(ServiceStateFailed)
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

		s.setState(ServiceStateStarting)
		select {
		case <-time.After(delay):
		case <-s.opts.Context.Done():
			return nil
		}
	}
}

// closeReady closes the readiness channel exactly once.
func (s *Service) closeReady() {
	select {
	case <-s.readyCh:
	default:
		close(s.readyCh)
	}
}

// runAfterStart runs the AfterStart hooks. On the first hook error it marks the
// service failed and returns that error. Called once, right after the service
// first becomes ready.
func (s *Service) runAfterStart() error {
	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			s.setState(ServiceStateFailed)
			return err
		}
	}
	return nil
}

// awaitShutdown waits for the StartFn goroutine to finish after the context was
// cancelled, bounded by ShutdownTimeout, and always returns nil.
func (s *Service) awaitShutdown(errChan chan error, doneChan chan struct{}) error {
	select {
	case <-time.After(s.opts.ShutdownTimeout):
		s.opts.Logger.Infof("service [%s] was stopped by timeout", s.Name())
	case <-doneChan:
	case <-errChan:
	}
	return nil
}

func (s *Service) Stop() error {
	if s.opts.StopFn == nil {
		return nil
	}

	if s.State() == ServiceStateStopped || s.State() == ServiceStateStopping || s.State() == ServiceStateIdle {
		return nil
	}
	s.setState(ServiceStateStopping)

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
		s.setState(ServiceStateStopped)
		s.opts.Logger.Infof("service [%s] was stopped", s.Name())
	case err := <-errChan:
		s.setState(ServiceStateFailed)
		return err
	case <-ctx.Done():
		s.setState(ServiceStateStopped)
		s.opts.Logger.Infof("failed to stop service [%s]. stop by timeout", s.Name())
	}

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	return stopErr
}
