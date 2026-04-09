//nolint:ireturn
package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"
	"time"

	"github.com/tkcrm/mx/launcher/ops"
	"golang.org/x/sync/errgroup"
)

type ILauncher interface {
	// Run launcher and all services
	Run() error
	// Stop launcher and all services
	Stop()
	// ServicesRunner return services runner
	ServicesRunner() IServicesRunner
	// Context return global context
	Context() context.Context

	// AddBeforeStartHooks adds before start hooks
	AddBeforeStartHooks(hook ...func() error)
	// AddBeforeStopHooks adds before stop hooks
	AddBeforeStopHooks(hook ...func() error)
	// AddAfterStartHooks adds after start hooks
	AddAfterStartHooks(hook ...func() error)
	// AddAfterStopHooks adds after stop hooks
	AddAfterStopHooks(hook ...func() error)
}

type launcher struct {
	opts Options

	cancelFn context.CancelFunc

	servicesRunner *servicesRunner
}

// New creates a new launcher.
func New(opts ...Option) ILauncher {
	l := &launcher{
		opts: newOptions(opts...),
	}

	ctx, cancel := context.WithCancel(l.opts.Context)
	l.opts.Context = ctx //nolint:fatcontext
	l.cancelFn = cancel

	l.servicesRunner = newServicesRunner(l.opts.Context, l.opts.logger)

	return l
}

// Run runs launcher and all services.
func (l *launcher) Run() error { //nolint:cyclop
	// register ops services
	if l.opts.OpsConfig.Enabled {
		if l.opts.OpsConfig.Healthy.Enabled {
			l.opts.OpsConfig.Healthy.AddServicesList(l.servicesRunner.hcServices())
			l.opts.OpsConfig.Healthy.AddStateList(l.servicesRunner.stateProviders())
		}
		opsSvcs := ops.New(l.opts.logger, l.opts.OpsConfig)
		svcs := make([]*Service, len(opsSvcs))
		for i := range opsSvcs {
			svcs[i] = NewService(WithService(opsSvcs[i]))
		}
		l.servicesRunner.Register(svcs...)
	}

	// before start
	for _, fn := range l.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	// group services by startup priority
	groups := make(map[int][]*Service)
	for _, svc := range l.servicesRunner.Services() {
		p := svc.Options().StartupPriority
		groups[p] = append(groups[p], svc)
	}

	// collect and sort unique priorities (excluding 0)
	var priorities []int
	for p := range groups {
		if p > 0 {
			priorities = append(priorities, p)
		}
	}
	slices.Sort(priorities)

	errChan := make(chan error, len(l.servicesRunner.Services()))
	graceWait := new(sync.WaitGroup)

	startSvc := func(svc *Service) {
		graceWait.Add(1)
		go func() {
			defer graceWait.Done()
			if err := svc.Start(); err != nil {
				errChan <- fmt.Errorf("failed to start service [%s]: %w", svc.Name(), err)
			}
		}()
	}

	// start priority groups sequentially; within each group — concurrently
	for _, p := range priorities {
		group := groups[p]

		for _, svc := range group {
			startSvc(svc)
		}

		// wait for ALL services in this group to become ready
		for _, svc := range group {
			select {
			case <-svc.Ready():
			case err := <-errChan:
				l.cancelFn()
				graceWait.Wait()
				return err
			case <-l.opts.Context.Done():
				graceWait.Wait()
				return l.opts.Context.Err()
			}
		}

		// Yield to let any immediate service failures propagate.
		// A service that fails synchronously closes readyCh (via deferred Start)
		// before the error reaches errChan. This sleep gives startSvc goroutines
		// time to deliver the error after Start() returns.
		time.Sleep(time.Nanosecond)
		select {
		case err := <-errChan:
			l.cancelFn()
			graceWait.Wait()
			return err
		default:
		}
	}

	// start priority-0 services — all concurrently
	for _, svc := range groups[0] {
		startSvc(svc)
	}

	if l.opts.AppStartStopLog {
		l.opts.logger.Infoln("app", l.opts.Name, "was started")
	}

	// after start
	for _, fn := range l.opts.AfterStart {
		if err := fn(); err != nil {
			l.cancelFn()
			graceWait.Wait()
			return err
		}
	}

	ch := make(chan os.Signal, 1)
	if l.opts.Signal {
		signal.Notify(ch, ShutdownSiganl()...)
		defer signal.Stop(ch)
	}

	var forceExitCancel context.CancelFunc

	select {
	// wait on services error
	case err := <-errChan:
		l.cancelFn()
		graceWait.Wait()
		return err
	// wait on kill signal
	case <-ch:
		l.cancelFn()
		l.opts.logger.Warnln("graceful shutdown started, send signal again to force exit")
		if l.opts.Signal {
			var forceCtx context.Context
			forceCtx, forceExitCancel = context.WithCancel(context.Background())
			go func() {
				select {
				case <-ch:
					l.opts.logger.Warnln("received second signal, forcing exit")
					os.Exit(1)
				case <-forceCtx.Done():
				}
			}()
		}
	// wait on context cancel
	case <-l.opts.Context.Done():
	}

	if forceExitCancel != nil {
		defer forceExitCancel()
	}

	graceWait.Wait()

	var stopCtx context.Context
	var stopCtxCancel context.CancelFunc
	if l.opts.GlobalShutdownTimeout > 0 {
		stopCtx, stopCtxCancel = context.WithTimeout(context.Background(), l.opts.GlobalShutdownTimeout)
	} else {
		stopCtx, stopCtxCancel = context.WithCancel(context.Background())
	}
	defer stopCtxCancel()

	// enforce global shutdown timeout
	go func() {
		<-stopCtx.Done()
		if stopCtx.Err() == context.DeadlineExceeded {
			l.opts.logger.Warnln("global shutdown timeout exceeded, forcing exit")
			os.Exit(1)
		}
	}()

	var stopErr error

	// before stop
	for _, fn := range l.opts.BeforeStop {
		if err := fn(); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
	}

	// stop services
	switch l.opts.RunnerServicesSequence {
	case RunnerServicesSequenceNone:
		{
			g := new(errgroup.Group)
			for i := range l.servicesRunner.Services() {
				svc := l.servicesRunner.Services()[i]
				g.Go(func() error {
					if err := svc.Stop(); err != nil {
						l.opts.logger.Errorf("failed to stop service [%s] error: %s", svc.Name(), err)
					}
					return nil
				})
			}

			// wait stop group
			if err := g.Wait(); err != nil {
				l.opts.logger.Errorf("failed to stop services: %s", err)
			}
		}
	case RunnerServicesSequenceFifo:
		{
			for _, svc := range l.servicesRunner.Services() {
				if err := svc.Stop(); err != nil {
					l.opts.logger.Errorf("failed to stop service [%s] error: %s", svc.Name(), err)
				}
			}
		}
	case RunnerServicesSequenceLifo:
		{
			reverted := make([]*Service, len(l.servicesRunner.Services()))
			copy(reverted, l.servicesRunner.Services())
			slices.Reverse(reverted)
			for _, svc := range reverted {
				if err := svc.Stop(); err != nil {
					l.opts.logger.Errorf("failed to stop service [%s] error: %s", svc.Name(), err)
				}
			}
		}
	}

	if l.opts.AppStartStopLog {
		l.opts.logger.Infoln("app", l.opts.Name, "was stopped")
	}

	// after stop
	for _, fn := range l.opts.AfterStop {
		if err := fn(); err != nil {
			stopErr = errors.Join(stopErr, err)
		}
	}

	return stopErr
}

// Stop stops launcher and all services.
func (l *launcher) Stop() { l.cancelFn() }

// ServicesRunner returns services runner.
func (l *launcher) ServicesRunner() IServicesRunner { return l.servicesRunner }

// Context returns global context.
func (l *launcher) Context() context.Context { return l.opts.Context }

// AddBeforeStartHooks adds before start hooks.
func (l *launcher) AddBeforeStartHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.BeforeStart = append(l.opts.BeforeStart, fn)
	}
}

// AddBeforeStopHooks adds before stop hooks.
func (l *launcher) AddBeforeStopHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.BeforeStop = append(l.opts.BeforeStop, fn)
	}
}

// AddAfterStartHooks adds after start hooks.
func (l *launcher) AddAfterStartHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.AfterStart = append(l.opts.AfterStart, fn)
	}
}

// AddAfterStopHooks adds after stop hooks.
func (l *launcher) AddAfterStopHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.AfterStop = append(l.opts.AfterStop, fn)
	}
}
