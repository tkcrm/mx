package service

import (
	"context"
	"fmt"
	"time"

	"github.com/tkcrm/mx/logger"
)

const (
	defaultServiceName = "unknown"
)

// Options for service
type Options struct {
	Logger logger.Logger

	Name          string
	Enabled       bool
	HealthChecker HealthChecker

	StartFn func(ctx context.Context) error
	StopFn  func(ctx context.Context) error

	// Before and After funcs
	BeforeStart        []func() error
	BeforeStop         []func() error
	AfterStart         []func() error
	AfterStartFinished []func() error
	AfterStop          []func() error

	Signal bool

	Context context.Context

	// Default 10 seconds
	ShutdownTimeout time.Duration
}

func (s *Options) Validate() error {
	if s.Logger == nil {
		return fmt.Errorf("undefined logger")
	}

	if s.Name == "" {
		return fmt.Errorf("empty name")
	}

	if s.StartFn == nil {
		return fmt.Errorf("undefined Start func")
	}

	if s.StopFn == nil {
		return fmt.Errorf("undefined Stop func")
	}

	if s.Context == nil {
		return fmt.Errorf("undefined context")
	}

	return nil
}

func newOptions(opts ...Option) Options {
	opt := Options{
		Logger: logger.New(),

		Name:    defaultServiceName,
		Enabled: true,

		BeforeStart:        make([]func() error, 0),
		BeforeStop:         make([]func() error, 0),
		AfterStart:         make([]func() error, 0),
		AfterStartFinished: make([]func() error, 0),
		AfterStop:          make([]func() error, 0),

		Signal: true,

		ShutdownTimeout: time.Second * 10,
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

type Option func(o *Options)

// HandleSignal toggles automatic installation of the signal handler that
// traps TERM, INT, and QUIT.  Users of this feature to disable the signal
// handler, should control liveness of the service through the context.
func WithSignal(b bool) Option {
	return func(o *Options) { o.Signal = b }
}

// Name of the service
func WithName(n string) Option {
	return func(o *Options) { o.Name = n }
}

func WithContext(ctx context.Context) Option {
	return func(o *Options) { o.Context = ctx }
}

func WithLogger(l logger.Logger) Option {
	return func(o *Options) { o.Logger = l }
}

func WithStart(fn func(context.Context) error) Option {
	return func(o *Options) { o.StartFn = fn }
}

func WithStop(fn func(context.Context) error) Option {
	return func(o *Options) { o.StopFn = fn }
}

func WithEnabled(v bool) Option {
	return func(o *Options) { o.Enabled = v }
}

func WithShutdownTimeout(v time.Duration) Option {
	return func(o *Options) { o.ShutdownTimeout = v }
}

func WithService(svc any) Option {
	return func(o *Options) {
		if impl, ok := svc.(interface{ Name() string }); ok {
			o.Name = impl.Name()
		}

		if impl, ok := svc.(interface{ Start(context.Context) error }); ok {
			o.StartFn = impl.Start
		}

		if impl, ok := svc.(interface{ Stop(context.Context) error }); ok {
			o.StopFn = impl.Stop
		}

		if impl, ok := svc.(Enabler); ok {
			o.Enabled = impl.Enabled()
		}

		if impl, ok := svc.(HealthChecker); ok {
			o.HealthChecker = impl
		}
	}
}

// Before and Afters

// WithBeforeStart run funcs before service starts
func WithBeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// WithBeforeStop run funcs before service stops
func WithBeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// WithAfterStart run funcs after service starts
func WithAfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// WithAfterStartFinished run funcs after was finished service start func
func WithAfterStartFinished(fn func() error) Option {
	return func(o *Options) {
		o.AfterStartFinished = append(o.AfterStart, fn)
	}
}

// WithAfterStop run funcs after service stops
func WithAfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}
