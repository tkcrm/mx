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
- [x] Ping pong service
- [x] Http transport
- [x] GRPC transport
- [x] GRPC client
- [x] ConnectRPC transport
- [x] ConnectRPC client

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

### Graceful shutdown

The first signal (SIGTERM / SIGINT / SIGQUIT) starts a graceful shutdown. A second signal forces immediate exit.

```go
if err := ln.Run(); err != nil {
    logger.Fatal(err)
}
```
