# Basic Service

A minimal service implementing `types.IService`. Place in `internal/services/{PACKAGE_NAME}/service.go`.

## service.go

```go
package {PACKAGE_NAME}

import (
	"context"

	"github.com/tkcrm/mx/launcher/types"
	"github.com/tkcrm/mx/logger"
)

type {SERVICE_STRUCT} struct {
	logger logger.Logger
}

func New(l logger.Logger) *{SERVICE_STRUCT} {
	return &{SERVICE_STRUCT}{logger: l}
}

// Name returns the service name used in logs and health checks.
func (s *{SERVICE_STRUCT}) Name() string { return "{SERVICE_NAME}" }

// Start runs the service. It must block until ctx is cancelled or work is done.
func (s *{SERVICE_STRUCT}) Start(ctx context.Context) error {
	s.logger.Infof("[%s] started", s.Name())

	// Do your work here. Block until context is cancelled.
	<-ctx.Done()

	return nil
}

// Stop gracefully shuts down the service. The ctx has a deadline
// equal to ShutdownTimeout (default 10s).
func (s *{SERVICE_STRUCT}) Stop(_ context.Context) error {
	return nil
}

// Compile-time check that {SERVICE_STRUCT} satisfies IService.
var _ types.IService = (*{SERVICE_STRUCT})(nil)
```

> Note: The `Start` function MUST block. If it returns immediately, the launcher treats the service as exited. Use `<-ctx.Done()` for long-running services, or run a server (HTTP/gRPC) that blocks until context cancellation.

## Inline Service (without a separate struct)

For simple cases, you can define a service inline using functional options:

```go
svc := launcher.NewService(
	launcher.WithServiceName("{SERVICE_NAME}"),
	launcher.WithStart(func(ctx context.Context) error {
		// service logic
		<-ctx.Done()
		return nil
	}),
	launcher.WithStop(func(ctx context.Context) error {
		// cleanup logic
		return nil
	}),
)

ln.ServicesRunner().Register(svc)
```
