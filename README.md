# MX

A Go microservices framework with runtime launcher and services runner

## Features

- [x] Logger
- [x] Launcher
- [x] Services
- [x] Services runner
- [x] Service `Enabler` interface
- [x] Service `HealthChecker` interface
- [x] Metrics
- [x] Health checker
- [x] Ping pong service
- [x] GRPC transport
- [x] Http transport
- [x] ConnectRPC transport
- [x] Config loader
- [x] CLI tools

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
svc := service.New(
    service.WithName("test-service"),
    service.WithStart(func(_ context.Context) error {
        return nil
    }),
    service.WithStop(func(_ context.Context) error {
        time.Sleep(time.Second * 3)
        return nil
    }),
)

// register in launcher
ln.ServicesRunner().Register(svc)
```

### Init and register ping pong service

```go
// init
pingPongSvc := service.New(service.WithService(pingpong.New(logger)))

// register in launcher
ln.ServicesRunner().Register(pingPongSvc)
```

You can also register any service that implements the following interface

```go
type IService interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

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

var _ service.HealthChecker = (*books)(nil)

var _ service.IService = (*books)(nil)

func main() {
    ln := launcher.New()

    // register service in launcher with health checker
    ln.ServicesRunner().Register(
        service.New(
            service.WithService(New()),
        ),
    )
}
```

### Start launcher and all services with graceful shutdown

```go
if err := ln.Run(); err != nil {
    logger.Fatal(err)
}
```

## Config loader

```go
type Config struct {
    ServiceName string            `default:"mx-example" validate:"required"`
    Prometheus  prometheus.Config
    Ops         ops.Config
    Grpc        grpc_transport.Config
}

conf := new(Config)
if err := cfg.Load(conf, cfg.WithVersion(version)); err != nil {
    logger.Fatalf("could not load configuration: %s", err)
}
```

### CLI commands

```bash
# Print help
go run main.go --help

# Print app version
go run main.go --version

# Validate application configuration without starting a microservice
go run main.go --validate

# Print markdown enviroment variables
go run main.go --markdown

# Print markdown enviroment variables and save to file
go run main.go --markdown --file ENVS.md
```
