package launcher

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/tkcrm/micro/service"
	signalutil "github.com/tkcrm/micro/util/signal"
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
}

type launcher struct {
	opts Options

	stop     chan struct{}
	cancelFn context.CancelFunc

	servicesRunner *servicesRunner
}

func New(opts ...Option) ILauncher {
	l := &launcher{
		opts: newOptions(opts...),
		stop: make(chan struct{}, 1),
	}

	ctx, cancel := context.WithCancel(l.opts.Context)
	l.opts.Context = ctx
	l.cancelFn = cancel

	l.servicesRunner = newServicesRunner(l.opts.Context, l.opts.logger)

	return l
}

func (l *launcher) Run() error {
	// before start
	for _, fn := range l.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	// start service
	var errChan = make(chan error, len(l.servicesRunner.Services()))
	for i := range l.servicesRunner.Services() {
		go func(svc *service.Service) {
			if err := svc.Start(); err != nil {
				err := fmt.Errorf("failed to start service [%s]: %w", svc.Name(), err)
				errChan <- err
			}
		}(l.servicesRunner.Services()[i])
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
	// wait on stop func
	case <-l.stop:
	// wait on kill signal
	case <-ch:
	// wait on context cancel
	case <-l.opts.Context.Done():
	}

	var stopErr error

	// before stop
	for _, fn := range l.opts.BeforeStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	// stop services
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
		return err
	}

	// after stop
	for _, fn := range l.opts.AfterStop {
		if err := fn(); err != nil {
			stopErr = err
		}
	}

	return stopErr
}

func (l *launcher) Stop() {
	l.stop <- struct{}{}
	l.cancelFn()
}

func (l *launcher) ServicesRunner() IServicesRunner {
	return l.servicesRunner
}

func (l *launcher) Context() context.Context {
	return l.opts.Context
}
