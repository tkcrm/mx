# Health Checker Service

A service that implements both `types.IService` and `types.HealthChecker`. The ops health endpoint will periodically call `Healthy()` and report results on `/healthy`, `/livez`, and `/readyz`.

## service.go

```go
package {PACKAGE_NAME}

import (
	"context"
	"fmt"
	"time"

	"github.com/tkcrm/mx/launcher/types"
	"github.com/tkcrm/mx/logger"
)

type {SERVICE_STRUCT} struct {
	logger logger.Logger
	ready  bool
}

func New(l logger.Logger) *{SERVICE_STRUCT} {
	return &{SERVICE_STRUCT}{logger: l}
}

func (s *{SERVICE_STRUCT}) Name() string { return "{SERVICE_NAME}" }

func (s *{SERVICE_STRUCT}) Start(ctx context.Context) error {
	// Perform initialization...
	s.ready = true

	<-ctx.Done()
	return nil
}

func (s *{SERVICE_STRUCT}) Stop(_ context.Context) error {
	s.ready = false
	return nil
}

// Interval returns how often the health check runs.
func (s *{SERVICE_STRUCT}) Interval() time.Duration {
	return 5 * time.Second
}

// Healthy returns nil if the service is healthy, or an error describing the problem.
// Return ops.ErrHealthCheckServiceStarting during initialization phase.
func (s *{SERVICE_STRUCT}) Healthy(_ context.Context) error {
	if !s.ready {
		return fmt.Errorf("service not ready")
	}
	return nil
}

var _ types.IService = (*{SERVICE_STRUCT})(nil)
var _ types.HealthChecker = (*{SERVICE_STRUCT})(nil)
```

> Note: When using `launcher.WithService(svc)`, the framework automatically detects the `HealthChecker` interface via duck-typing and registers it with the ops health checker. No extra wiring is needed — just implement the interface and register with `launcher.NewService(launcher.WithService(svc))`.

## Health Check Response Codes

The ops health endpoint returns:

| HTTP Status             | Meaning                          |
| ----------------------- | -------------------------------- |
| 200 OK                  | All health checks pass           |
| 424 Failed Dependency   | A service is still starting      |
| 503 Service Unavailable | A health check returned an error |

## Liveness vs Readiness

- **`/livez` (liveness)**: Checks `ServiceState` only. Returns 503 if any service is in `Failed` state. Does NOT run `HealthChecker`.
- **`/readyz` (readiness)**: Combines `ServiceState` + `HealthChecker` results. Returns 200 only when all services are `Running` AND all health checks pass.
