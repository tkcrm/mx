# MX

[![Go Reference](https://pkg.go.dev/badge/github.com/tkcrm/mx.svg)](https://pkg.go.dev/github.com/tkcrm/mx)
[![Go Version](https://img.shields.io/badge/go-1.25-blue)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/tkcrm/mx)](https://goreportcard.com/report/github.com/tkcrm/mx)
[![License](https://img.shields.io/github/license/tkcrm/mx)](LICENSE)

A Go microservices framework with runtime launcher and services runner

## Features

- [x] Logger
- [x] Launcher
- [x] Services
- [x] Services runner
- [x] `Enabler` interface
- [x] `HealthChecker` interface
- [x] Metrics
- [x] Health checker
- [x] Liveness probe (`/livez`)
- [x] Readiness probe (`/readyz`)
- [x] Ping pong service
- [x] Http transport
- [x] GRPC transport
- [x] GRPC client
- [x] ConnectRPC transport
- [x] ConnectRPC client

## AI Agent Skills

This repository includes [AI agent skills](https://github.com/sxwebdev/skills) with documentation and usage examples for all packages. Install them with the [skills](https://github.com/sxwebdev/skills) CLI:

```bash
go install github.com/sxwebdev/skills/cmd/skills@latest
skills init
skills repo add tkcrm/mx
```

## Launcher capabilities

| Capability                     | Option / Interface                                                     | Description                                                                                     |
| ------------------------------ | ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Lifecycle hooks                | `WithBeforeStart`, `WithAfterStart`, `WithBeforeStop`, `WithAfterStop` | Global hooks around app start/stop                                                              |
| Service state machine          | `svc.State()`                                                          | Tracks each service: `idle → starting → running → stopping → stopped / failed`                  |
| Service restart policy         | `WithRestartPolicy(RestartPolicy{...})`                                | `RestartOnFailure` / `RestartAlways` with exponential backoff                                   |
| Startup timeout                | `WithStartupTimeout(d)`                                                | Fail the service if `StartFn` does not signal ready within `d`                                  |
| Shutdown timeout (per service) | `WithShutdownTimeout(d)`                                               | Max time to wait for a service to stop                                                          |
| Global shutdown timeout        | `WithGlobalShutdownTimeout(d)`                                         | Hard deadline for the entire graceful shutdown phase                                            |
| Startup priority               | `WithStartupPriority(n)`                                               | Group-based startup ordering: same priority starts concurrently, groups run in ascending order  |
| Stop sequence                  | `WithRunnerServicesSequence(...)`                                      | `None` (parallel) / `Fifo` / `Lifo`                                                             |
| Service lookup                 | `ServicesRunner().Get(name)`                                           | Retrieve a registered service by name at runtime                                                |
| Health checker                 | `types.HealthChecker` interface                                        | Periodic per-service health check, polled on a configurable interval                            |
| Liveness probe                 | ops `/livez`                                                           | `200` healthy / `503` if any service is in `Failed` state                                       |
| Readiness probe                | ops `/readyz`                                                          | `200` ready / `424` starting / `503` failed — combines `ServiceState` + `HealthChecker` results |
| Legacy health endpoint         | ops `/healthy`                                                         | Backward-compatible endpoint (HealthChecker results only)                                       |
| Metrics                        | ops `/metrics`                                                         | Prometheus metrics endpoint                                                                     |
| Profiler                       | ops `/debug/pprof`                                                     | Go pprof profiler endpoint                                                                      |

## How to use

Repo with [example](https://github.com/tkcrm/mx-example)

### Init launcher

```go
var version = "local"
var appName = "mx-example"

logger := logger.New(
    logger.WithAppVersion(version),
    logger.WithAppName(appName),
)

ln := launcher.New(
    launcher.WithName(appName),
    launcher.WithLogger(logger),
    launcher.WithVersion(version),
    launcher.WithContext(context.Background()),
    launcher.WithAfterStart(func() error {
        logger.Infoln("app", appName, "was started")
        return nil
    }),
    launcher.WithAfterStop(func() error {
        logger.Infoln("app", appName, "was stopped")
        return nil
    }),
)
```

### Init and register custom service

```go
// init
svc := launcher.NewService(
    launcher.WithServiceName("test-service"),
    launcher.WithStart(func(_ context.Context) error {
        return nil
    }),
    launcher.WithStop(func(_ context.Context) error {
        time.Sleep(time.Second * 3)
        return nil
    }),
)

// register in launcher
ln.ServicesRunner().Register(svc)
```

### Init and register ping pong service

```go
import "github.com/tkcrm/mx/launcher/services/pingpong"

// init
pingPongSvc := launcher.NewService(launcher.WithService(pingpong.New(logger)))

// register in launcher
ln.ServicesRunner().Register(pingPongSvc)
```

### Register any service that implements IService

Any struct with `Name()`, `Start()`, and `Stop()` methods satisfies `types.IService` and can be wrapped with `launcher.NewService`:

```go
import "github.com/tkcrm/mx/launcher/types"

type books struct {
    name       string
    hcInterval time.Duration
}

func New() *books {
    return &books{
        name:       "books-service",
        hcInterval: time.Second * 3,
    }
}

func (s books) Name() string { return s.name }

func (s books) Healthy(ctx context.Context) error { return nil }

func (s books) Interval() time.Duration { return s.hcInterval }

func (s books) Start(ctx context.Context) error {
    <-ctx.Done()
    return nil
}

func (s books) Stop(ctx context.Context) error { return nil }

var _ types.HealthChecker = (*books)(nil)
var _ types.IService = (*books)(nil)

func main() {
    ln := launcher.New()

    // register service in launcher with health checker
    ln.ServicesRunner().Register(
        launcher.NewService(
            launcher.WithService(New()),
        ),
    )
}
```

### Startup priority

Services can be assigned a startup priority to control initialization order. Services with the same priority start concurrently within a group. Groups are started sequentially in ascending priority order. Priority 0 (default) services start last, concurrently, after all prioritized groups are ready.

```go
ln.ServicesRunner().Register(
    // Priority 1: DB layer — start concurrently, both must be ready before next group
    launcher.NewService(
        launcher.WithServiceName("postgres"),
        launcher.WithStartupPriority(1),
        launcher.WithService(pgService),
    ),
    launcher.NewService(
        launcher.WithServiceName("redis"),
        launcher.WithStartupPriority(1),
        launcher.WithService(redisService),
    ),
    // Priority 2: message broker — waits for DB layer to be ready
    launcher.NewService(
        launcher.WithServiceName("rabbitmq"),
        launcher.WithStartupPriority(2),
        launcher.WithService(rabbitService),
    ),
    // Priority 0 (default): application services — start concurrently after all groups
    launcher.NewService(
        launcher.WithServiceName("http-server"),
        launcher.WithService(httpService),
    ),
    launcher.NewService(
        launcher.WithServiceName("grpc-server"),
        launcher.WithService(grpcService),
    ),
)
// Start order: (postgres + redis) → rabbitmq → (http + grpc concurrently)
```

### Graceful shutdown

The first signal (SIGTERM / SIGINT / SIGQUIT) starts a graceful shutdown. A second signal forces immediate exit.

```go
if err := ln.Run(); err != nil {
    logger.Fatal(err)
}
```
