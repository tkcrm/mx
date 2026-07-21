package launcher

import (
	"context"
	"errors"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/mxtypes"
)

const defaultServiceName = "unknown"

// ServiceOptions holds configuration for a Service.
type ServiceOptions struct {
	Logger logger.Logger

	Name          string
	Enabled       bool
	HealthChecker mxtypes.HealthChecker

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

	// Readiness, when non-nil, returns a channel that is closed once the service
	// has finished starting up and is operational. The launcher keeps the service
	// in the Starting state until it closes, gates startup-priority groups on it,
	// and bounds the wait for it with StartupTimeout. Nil means the service is
	// considered ready as soon as its Start goroutine is launched.
	Readiness func() <-chan struct{}

	// StartupTimeout is the maximum time to wait for a service that reports
	// readiness (see Readiness) to become ready. A service that does not report
	// ready within this duration is marked failed; a service that reports ready
	// is never killed by this timeout, however long it subsequently runs.
	// Zero means no startup timeout. It has no effect on services that do not
	// report readiness — they are ready immediately.
	StartupTimeout time.Duration

	// RestartPolicy defines how the service behaves after an unexpected exit.
	RestartPolicy RestartPolicy

	// StartupPriority controls service startup ordering.
	// Services are grouped by priority and started group-by-group in ascending order.
	// Services within the same priority group start concurrently.
	// All services in a group must reach Running state before the next group starts.
	// Priority 0 (default): start concurrently after all prioritized groups are ready.
	StartupPriority int
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

// WithStartupTimeout sets the maximum time to wait for a service that reports
// readiness to become ready before it is considered failed. Zero (default)
// means no startup timeout. It has no effect on services that do not report
// readiness (see WithReadiness / mxtypes.ReadinessReporter).
func WithStartupTimeout(v time.Duration) ServiceOption {
	return func(o *ServiceOptions) { o.StartupTimeout = v }
}

// WithReadiness sets a readiness signal for the service. The returned channel
// must be closed once the service is operational. The launcher keeps the
// service in the Starting state until it closes, gates startup-priority groups
// on it, and bounds the wait with StartupTimeout. Use this for inline services
// defined via WithStart; struct services can instead implement
// mxtypes.ReadinessReporter and be wrapped with WithService.
func WithReadiness(ch <-chan struct{}) ServiceOption {
	return func(o *ServiceOptions) {
		o.Readiness = func() <-chan struct{} { return ch }
	}
}

// WithRestartPolicy configures automatic restart behaviour after an unexpected exit.
func WithRestartPolicy(p RestartPolicy) ServiceOption {
	return func(o *ServiceOptions) { o.RestartPolicy = p }
}

// WithStartupPriority sets the startup priority for the service.
// Services with the same priority start concurrently within a group.
// Groups are started sequentially in ascending priority order.
// Priority 0 (default) services start last, concurrently.
func WithStartupPriority(p int) ServiceOption {
	return func(o *ServiceOptions) { o.StartupPriority = p }
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

		if impl, ok := svc.(mxtypes.Enabler); ok {
			o.Enabled = impl.Enabled()
		}

		if impl, ok := svc.(mxtypes.HealthChecker); ok {
			o.HealthChecker = impl
		}

		if impl, ok := svc.(mxtypes.ReadinessReporter); ok {
			o.Readiness = impl.Ready
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
