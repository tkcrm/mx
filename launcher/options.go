package launcher

import (
	"context"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/ops"
)

type Option func(*Options)

type Options struct {
	logger logger.ExtendedLogger

	Name    string
	Version string

	// Before and After funcs
	BeforeStart []func() error
	BeforeStop  []func() error
	AfterStart  []func() error
	AfterStop   []func() error

	AppStartStopLog bool

	RunnerServicesSequence RunnerServicesSequence

	Signal bool

	Context context.Context //nolint:containedctx

	OpsConfig ops.Config
}

func newOptions(opts ...Option) Options {
	opt := Options{
		logger: logger.NewExtended(),

		BeforeStart: make([]func() error, 0),
		BeforeStop:  make([]func() error, 0),
		AfterStart:  make([]func() error, 0),
		AfterStop:   make([]func() error, 0),

		RunnerServicesSequence: RunnerServicesSequenceNone,

		Signal: true,

		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

// Name of the launcher.
func WithName(n string) Option {
	return func(o *Options) { o.Name = n }
}

// Version of the launcher.
func WithVersion(v string) Option {
	return func(o *Options) { o.Version = v }
}

func WithContext(ctx context.Context) Option {
	return func(o *Options) { o.Context = ctx }
}

func WithRunnerServicesSequence(v RunnerServicesSequence) Option {
	return func(o *Options) { o.RunnerServicesSequence = v }
}

func WithSignal(b bool) Option {
	return func(o *Options) { o.Signal = b }
}

func WithLogger(l logger.ExtendedLogger) Option {
	return func(o *Options) { o.logger = l }
}

func WithOpsConfig(c ops.Config) Option {
	return func(o *Options) { o.OpsConfig = c }
}

func WithAppStartStopLog(v bool) Option {
	return func(o *Options) { o.AppStartStopLog = v }
}

// Before and Afters

// WithBeforeStart run funcs before service starts.
func WithBeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// WithBeforeStop run funcs before service stops.
func WithBeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// WithAfterStart run funcs after service starts.
func WithAfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// WithAfterStop run funcs after service stops.
func WithAfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}
