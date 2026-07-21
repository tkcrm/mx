package mxtypes

import (
	"context"
	"time"
)

// IService is the interface that wraps the basic service lifecycle methods.
type IService interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HealthChecker provides functionality to check health of any entity
// that implements this interface.
type HealthChecker interface {
	Name() string
	Interval() time.Duration
	Healthy(ctx context.Context) error
}

// Enabler is the interface that provides enabled state of a service.
type Enabler interface {
	Enabled() bool
}

// ReadinessReporter is an optional interface a service can implement to signal
// when it has finished starting up and is operational (e.g. a database has
// connected, an HTTP server is listening).
//
// When a service implements it, the launcher:
//   - keeps the service in the Starting state until the returned channel is
//     closed, only then transitioning it to Running;
//   - gates startup-priority groups on this signal, so the next group (and the
//     default priority-0 group) starts only once every service in the current
//     group has reported ready;
//   - applies StartupTimeout to the wait for this signal — a service that does
//     not report ready within StartupTimeout fails, but one that reports ready
//     is never killed no matter how long it then runs.
//
// The channel must be closed (not sent to) exactly once, when the service is
// ready. Services that do not implement this interface are considered ready as
// soon as their Start goroutine is launched, and StartupTimeout does not apply
// to them.
type ReadinessReporter interface {
	Ready() <-chan struct{}
}

// ServiceState represents the current lifecycle state of a service.
type ServiceState int

const (
	// ServiceStateIdle means the service has been registered but not started.
	ServiceStateIdle ServiceState = iota
	// ServiceStateStarting means the service Start func is running.
	ServiceStateStarting
	// ServiceStateRunning means the service started successfully and is running.
	ServiceStateRunning
	// ServiceStateStopping means the service Stop func is running.
	ServiceStateStopping
	// ServiceStateStopped means the service stopped cleanly.
	ServiceStateStopped
	// ServiceStateFailed means the service exited with an error.
	ServiceStateFailed
)

func (s ServiceState) String() string {
	switch s {
	case ServiceStateIdle:
		return "idle"
	case ServiceStateStarting:
		return "starting"
	case ServiceStateRunning:
		return "running"
	case ServiceStateStopping:
		return "stopping"
	case ServiceStateStopped:
		return "stopped"
	case ServiceStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// StateProvider exposes the current lifecycle state of a named component.
type StateProvider interface {
	Name() string
	State() ServiceState
}
