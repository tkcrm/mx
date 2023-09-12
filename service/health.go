package service

import (
	"context"
	"time"
)

// HealthChecker provides functionality to check health of any entity
// that implement this interface.
type HealthChecker interface {
	Name() string
	Interval() time.Duration
	Healthy(ctx context.Context) error
}
