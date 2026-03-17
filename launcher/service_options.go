package launcher

import (
	"context"
	"errors"
	"time"

	"github.com/tkcrm/mx/launcher/types"
	"github.com/tkcrm/mx/logger"
)

const defaultServiceName = "unknown"

// ServiceOptions holds configuration for a Service.
type ServiceOptions struct {
	Logger logger.Logger

	Name          string
	Enabled       bool
	HealthChecker types.HealthChecker

	StartFn func(ctx context.Context) error
	StopFn  func(ctx context.Context) error

	// Before and After funcs
	BeforeStart        []func() error
	BeforeStop         []func() error
	AfterStart         []func() error
	AfterStartFinished []func() error
	AfterStop          []func() error

	Context context.Context //nolint:containedctx

	// ShutdownTimeout is the maximum time to wait for Stop to complete. Default 10 seconds.
	ShutdownTimeout time.Duration

	// StartupTimeout is the maximum time to wait for the StartFn goroutine to signal an error
	// after launch. Zero means no per-service startup timeout (the service runs until context cancel).
	StartupTimeout time.Duration

	// RestartPolicy defines how the service behaves after an unexpected exit.
	RestartPolicy RestartPolicy
}

func (s *ServiceOptions) Validate() error {
	if s.Logger == nil {
		return errors.New("undefined logger")
	}

	if s.Name == "" {
		return errors.New("empty name")
	}

	if s.StartFn == nil {
		return errors.New("undefined Start func")
	}

	if s.StopFn == nil {
		return errors.New("undefined Stop func")
	}

	if s.Context == nil {
		return errors.New("undefined context")
	}

	return nil
}

func newServiceOptions(opts ...ServiceOption) ServiceOptions {
	opt := ServiceOptions{
		Logger: logger.New(),

		Name:    defaultServiceName,
		Enabled: true,

		BeforeStart:        make([]func() error, 0),
		BeforeStop:         make([]func() error, 0),
		AfterStart:         make([]func() error, 0),
		AfterStartFinished: make([]func() error, 0),
		AfterStop:          make([]func() error, 0),

		ShutdownTimeout: time.Second * 10,
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// ServiceOption is a function that configures a ServiceOptions.
type ServiceOption func(o *ServiceOptions)

// WithServiceName sets the name of the service.
func WithServiceName(n string) ServiceOption {
	return func(o *ServiceOptions) { o.Name = n }
}

func WithServiceContext(ctx context.Context) ServiceOption {
	return func(o *ServiceOptions) { o.Context = ctx }
}

func WithServiceLogger(l logger.Logger) ServiceOption {
	return func(o *ServiceOptions) { o.Logger = l }
}

// WithStart sets the start function of the service.
func WithStart(fn func(context.Context) error) ServiceOption {
	return func(o *ServiceOptions) { o.StartFn = fn }
}

// WithStop sets the stop function of the service.
func WithStop(fn func(context.Context) error) ServiceOption {
	return func(o *ServiceOptions) { o.StopFn = fn }
}

// WithEnabled sets the enabled state of the service.
func WithEnabled(v bool) ServiceOption {
	return func(o *ServiceOptions) { o.Enabled = v }
}

// WithShutdownTimeout sets the shutdown timeout of the service.
func WithShutdownTimeout(v time.Duration) ServiceOption {
	return func(o *ServiceOptions) { o.ShutdownTimeout = v }
}

// WithStartupTimeout sets the maximum time the service's StartFn is allowed to
// run before being considered failed. Zero (default) means no startup timeout.
func WithStartupTimeout(v time.Duration) ServiceOption {
	return func(o *ServiceOptions) { o.StartupTimeout = v }
}

// WithRestartPolicy configures automatic restart behaviour after an unexpected exit.
func WithRestartPolicy(p RestartPolicy) ServiceOption {
	return func(o *ServiceOptions) { o.RestartPolicy = p }
}

// WithService wraps any value that implements Name/Start/Stop/Enabled/HealthChecker.
func WithService(svc any) ServiceOption {
	return func(o *ServiceOptions) {
		if impl, ok := svc.(interface{ Name() string }); ok {
			o.Name = impl.Name()
		}

		if impl, ok := svc.(interface{ Start(_ context.Context) error }); ok {
			o.StartFn = impl.Start
		}

		if impl, ok := svc.(interface{ Stop(_ context.Context) error }); ok {
			o.StopFn = impl.Stop
		}

		if impl, ok := svc.(types.Enabler); ok {
			o.Enabled = impl.Enabled()
		}

		if impl, ok := svc.(types.HealthChecker); ok {
			o.HealthChecker = impl
		}
	}
}

// WithServiceBeforeStart runs fn before service starts.
func WithServiceBeforeStart(fn func() error) ServiceOption {
	return func(o *ServiceOptions) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// WithServiceBeforeStop runs fn before service stops.
func WithServiceBeforeStop(fn func() error) ServiceOption {
	return func(o *ServiceOptions) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// WithServiceAfterStart runs fn after service starts.
func WithServiceAfterStart(fn func() error) ServiceOption {
	return func(o *ServiceOptions) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// WithServiceAfterStartFinished runs fn after the service Start func finishes.
func WithServiceAfterStartFinished(fn func() error) ServiceOption {
	return func(o *ServiceOptions) {
		o.AfterStartFinished = append(o.AfterStartFinished, fn)
	}
}

// WithServiceAfterStop runs fn after service stops.
func WithServiceAfterStop(fn func() error) ServiceOption {
	return func(o *ServiceOptions) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}
