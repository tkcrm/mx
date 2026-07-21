# Service Checklist

Step-by-step guide for adding a new service to an MX-based application.

## 1. Create the service struct

Create a new package under `internal/services/{SERVICE_NAME}/`:

```go
package {PACKAGE_NAME}

import (
    "context"
    "github.com/tkcrm/mx/mxtypes"
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
var _ mxtypes.IService = (*{SERVICE_STRUCT})(nil)
```

## 2. (Optional) Add health checking

If the service needs health monitoring, implement `mxtypes.HealthChecker`:

```go
func (s *{SERVICE_STRUCT}) Interval() time.Duration {
    return 5 * time.Second
}

func (s *{SERVICE_STRUCT}) Healthy(ctx context.Context) error {
    // Return nil if healthy, error if not
    return nil
}

var _ mxtypes.HealthChecker = (*{SERVICE_STRUCT})(nil)
```

## 3. (Optional) Add enable/disable support

Implement `mxtypes.Enabler` for conditional service registration:

```go
func (s *{SERVICE_STRUCT}) Enabled() bool {
    return s.config.Enabled
}

var _ mxtypes.Enabler = (*{SERVICE_STRUCT})(nil)
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

## 6. (Optional) Set startup priority

For infrastructure services that must be ready before other services start:

```go
ln.ServicesRunner().Register(
    // Priority 1 services start first (same priority = concurrent within group)
    launcher.NewService(
        launcher.WithService(dbSvc),
        launcher.WithStartupPriority(1),
    ),
    launcher.NewService(
        launcher.WithService(redisSvc),
        launcher.WithStartupPriority(1),
    ),
    // Priority 0 (default) starts last, after all prioritized groups are ready
    launcher.NewService(
        launcher.WithService(appSvc),
    ),
)
```

**Important — report readiness for real gating.** A priority group blocks the
next group until every service in it is *ready*. A service is "ready" only when
it says so: implement `mxtypes.ReadinessReporter` (`Ready() <-chan struct{}`,
close the channel once connected/listening) or pass `launcher.WithReadiness(ch)`
for an inline service. A service that does **not** report readiness is treated
as ready the instant its `Start` goroutine is launched — so without it, a
priority-0 app can start before the priority-1 database has actually connected.
`WithStartupTimeout(d)` bounds the wait for this readiness signal (and only
applies to services that report it).

```go
type DBService struct {
    ready chan struct{}
    // ...
}

func (s *DBService) Ready() <-chan struct{} { return s.ready }

func (s *DBService) Start(ctx context.Context) error {
    if err := s.connect(ctx); err != nil {
        return err // fails before signalling ready → group start aborts
    }
    close(s.ready) // now the next startup-priority group may begin
    <-ctx.Done()
    return nil
}
```

## 7. (Optional) Add lifecycle hooks

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

## 8. Verify

- [ ] Service implements `mxtypes.IService` (Name, Start, Stop)
- [ ] Start function blocks until `ctx.Done()` or work completes
- [ ] Stop function completes within `ShutdownTimeout` (default 10s)
- [ ] Health checker returns meaningful errors (if implemented)
- [ ] Service is registered before `launcher.Run()` is called
