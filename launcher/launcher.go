package launcher

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"sync"

	"github.com/tkcrm/mx/ops"
	"github.com/tkcrm/mx/service"
	signalutil "github.com/tkcrm/mx/util/signal"
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

// New creates a new launcher
func New(opts ...Option) ILauncher {
	l := &launcher{
		opts: newOptions(opts...),
	}

	ctx, cancel := context.WithCancel(l.opts.Context)
	l.opts.Context = ctx
	l.cancelFn = cancel

	l.servicesRunner = newServicesRunner(l.opts.Context, l.opts.logger)

	return l
}

// Run runs launcher and all services
func (l *launcher) Run() error {
	// register ops services
	if l.opts.OpsConfig.Enabled {
		if l.opts.OpsConfig.Healthy.Enabled {
			l.opts.OpsConfig.Healthy.AddServicesList(l.servicesRunner.hcServices())
		}
		opsSvcs := ops.New(l.opts.logger, l.opts.OpsConfig)
		svcs := make([]*service.Service, len(opsSvcs))
		for i := range opsSvcs {
			svcs[i] = service.New(service.WithService(opsSvcs[i]))
		}
		l.servicesRunner.Register(svcs...)
	}

	// before start
	for _, fn := range l.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	// start service
	errChan := make(chan error, len(l.servicesRunner.Services()))
	graceWait := new(sync.WaitGroup)
	graceWait.Add(len(l.servicesRunner.Services()))
	for i := range l.servicesRunner.Services() {
		go func(svc *service.Service) {
			defer graceWait.Done()
			if err := svc.Start(); err != nil {
				err := fmt.Errorf("failed to start service [%s]: %w", svc.Name(), err)
				errChan <- err
			}
		}(l.servicesRunner.Services()[i])
	}

	if l.opts.AppStartStopLog {
		l.opts.logger.Infoln("app", l.opts.Name, "was started")
	}

	// after start
	for _, fn := range l.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	ch := make(chan os.Signal, 1)
	if l.opts.Signal {
		signal.Notify(ch, signalutil.Shutdown()...)
	}

	select {
	// wait on services error
	case err := <-errChan:
		return err
	// wait on kill signal
	case <-ch:
		l.cancelFn()
	// wait on context cancel
	case <-l.opts.Context.Done():
	}

	graceWait.Wait()

	var stopErr error

	// before stop
	for _, fn := range l.opts.BeforeStop {
		if err := fn(); err != nil {
			stopErr = err
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
			g.Wait()
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
			reverted := make([]*service.Service, len(l.servicesRunner.Services()))
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
			stopErr = err
		}
	}

	return stopErr
}

// Stop stops launcher and all services
func (l *launcher) Stop() { l.cancelFn() }

// ServicesRunner returns services runner
func (l *launcher) ServicesRunner() IServicesRunner { return l.servicesRunner }

// Context returns global context
func (l *launcher) Context() context.Context { return l.opts.Context }

// AddBeforeStartHooks adds before start hooks
func (l *launcher) AddBeforeStartHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.BeforeStart = append(l.opts.BeforeStart, fn)
	}
}

// AddBeforeStopHooks adds before stop hooks
func (l *launcher) AddBeforeStopHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.BeforeStop = append(l.opts.BeforeStop, fn)
	}
}

// AddAfterStartHooks adds after start hooks
func (l *launcher) AddAfterStartHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.AfterStart = append(l.opts.AfterStart, fn)
	}
}

// AddAfterStopHooks adds after stop hooks
func (l *launcher) AddAfterStopHooks(hook ...func() error) {
	for _, fn := range hook {
		if fn == nil {
			continue
		}
		l.opts.AfterStop = append(l.opts.AfterStop, fn)
	}
}
