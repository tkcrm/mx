package service

import (
	"context"

	"github.com/tkcrm/micro/logger"
)

const (
	defaultServiceName = "unknown"
)

// Options for micro service
type Options struct {
	logger logger.Logger

	Name    string
	Enabled bool

	Start func(ctx context.Context) error
	Stop  func(ctx context.Context) error

	// Before and After funcs
	BeforeStart        []func() error
	BeforeStop         []func() error
	AfterStart         []func() error
	AfterStartFinished []func() error
	AfterStop          []func() error

	Signal bool

	Context context.Context
}

func newOptions(opts ...Option) Options {
	opt := Options{
		logger: logger.New(),

		Name:    defaultServiceName,
		Enabled: true,

		BeforeStart:        make([]func() error, 0),
		BeforeStop:         make([]func() error, 0),
		AfterStart:         make([]func() error, 0),
		AfterStartFinished: make([]func() error, 0),
		AfterStop:          make([]func() error, 0),

		Signal: true,
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
	return func(o *Options) {
		o.Signal = b
	}
}

// Name of the service
func WithName(n string) Option {
	return func(o *Options) {
		o.Name = n
	}
}

func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

func WithLogger(l logger.Logger) Option {
	return func(o *Options) {
		o.logger = l
	}
}

func WithStart(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.Start = fn
	}
}

func WithStop(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.Stop = fn
	}
}

func WithEnabled(v bool) Option {
	return func(o *Options) {
		o.Enabled = v
	}
}

func WithService(svc any) Option {
	return func(o *Options) {
		if impl, ok := svc.(interface{ Name() string }); ok {
			o.Name = impl.Name()
		}

		if impl, ok := svc.(interface{ Start(context.Context) error }); ok {
			o.Start = impl.Start
		}

		if impl, ok := svc.(interface{ Stop(context.Context) error }); ok {
			o.Stop = impl.Stop
		}

		if impl, ok := svc.(Enabler); ok {
			o.Enabled = impl.Enabled()
		}
	}
}

// Before and Afters

// BeforeStart run funcs before service starts
func BeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// BeforeStop run funcs before service stops
func BeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// AfterStart run funcs after service starts
func AfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// AfterStartFinished run funcs after was finished service start func
func AfterStartFinished(fn func() error) Option {
	return func(o *Options) {
		o.AfterStartFinished = append(o.AfterStart, fn)
	}
}

// AfterStop run funcs after service stops
func AfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}