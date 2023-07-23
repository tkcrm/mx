# MX

A Go microservices framework with runtime launcher and services runner

## Features

- [x] Logger
- [x] Launcher
- [x] Services
- [x] Services runner
- [x] Service `Enabler` interface
- [ ] Service `Healthy` interface
- [ ] Metrics
- [ ] Health checker
- [x] Ping pong service
- [ ] GRPC transport
- [ ] Fiber transport
- [x] Config loader

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

type myService struct {}

func (s *myService) Name() string { return "my-service" }

func (s *myService) Start(ctx context.Context) error { return nil }

func (s *myService) Stop(ctx context.Context) error { return nil }

func main() {
    // init service
    svc := service.New(
        service.WithService(&myService{}),
    )

    // register service in launcher
    ln.ServicesRunner().Register(svc)
}
```

### Start launcher and all services with graceful shutdown

```go
if err := ln.Run(); err != nil {
    logger.Fatal(err)
}
```
