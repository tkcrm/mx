package types

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
