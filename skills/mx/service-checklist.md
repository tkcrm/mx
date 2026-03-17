# Service Checklist

Step-by-step guide for adding a new service to an MX-based application.

## 1. Create the service struct

Create a new package under `internal/services/{SERVICE_NAME}/`:

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

func (s *{SERVICE_STRUCT}) Name() string { return "{SERVICE_NAME}" }

func (s *{SERVICE_STRUCT}) Start(ctx context.Context) error {
    // Block until ctx is cancelled
    <-ctx.Done()
    return nil
}

func (s *{SERVICE_STRUCT}) Stop(_ context.Context) error {
    return nil
}

// Compile-time interface check
var _ types.IService = (*{SERVICE_STRUCT})(nil)
```

## 2. (Optional) Add health checking

If the service needs health monitoring, implement `types.HealthChecker`:

```go
func (s *{SERVICE_STRUCT}) Interval() time.Duration {
    return 5 * time.Second
}

func (s *{SERVICE_STRUCT}) Healthy(ctx context.Context) error {
    // Return nil if healthy, error if not
    return nil
}

var _ types.HealthChecker = (*{SERVICE_STRUCT})(nil)
```

## 3. (Optional) Add enable/disable support

Implement `types.Enabler` for conditional service registration:

```go
func (s *{SERVICE_STRUCT}) Enabled() bool {
    return s.config.Enabled
}

var _ types.Enabler = (*{SERVICE_STRUCT})(nil)
```

## 4. Register with the launcher

In your `main.go` or bootstrap function:

```go
svc := {PACKAGE_NAME}.New(log)

ln.ServicesRunner().Register(
    launcher.NewService(launcher.WithService(svc)),
)
```

## 5. (Optional) Configure restart policy

For services that should auto-restart:

```go
ln.ServicesRunner().Register(
    launcher.NewService(
        launcher.WithService(svc),
        launcher.WithRestartPolicy(launcher.RestartPolicy{
            Mode:       launcher.RestartOnFailure,
            MaxRetries: 5,
            Delay:      time.Second,
            MaxDelay:   30 * time.Second,
        }),
    ),
)
```

## 6. (Optional) Add lifecycle hooks

```go
ln.ServicesRunner().Register(
    launcher.NewService(
        launcher.WithService(svc),
        launcher.WithServiceBeforeStart(func() error {
            log.Info("preparing service...")
            return nil
        }),
        launcher.WithServiceAfterStop(func() error {
            log.Info("service cleanup complete")
            return nil
        }),
    ),
)
```

## 7. Verify

- [ ] Service implements `types.IService` (Name, Start, Stop)
- [ ] Start function blocks until `ctx.Done()` or work completes
- [ ] Stop function completes within `ShutdownTimeout` (default 10s)
- [ ] Health checker returns meaningful errors (if implemented)
- [ ] Service is registered before `launcher.Run()` is called
