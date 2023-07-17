# GO micro

A Go microservices framework with runtime launcher

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

## Hot to use

Repo with [example](https://github.com/tkcrm/micro-example)

### Init launcher

```go
var version = "local"
var appName = "micro-example"

logger := logger.New(
    logger.WithAppVersion(version),
    logger.WithAppName(appName),
)

ln := launcher.New(
    launcher.WithName(appName),
    launcher.WithLogger(logger),
    launcher.WithVersion(version),
    launcher.WithContext(context.Background()),
    launcher.AfterStart(func() error {
        logger.Infoln("service", appName, "started")
        return nil
    }),
    launcher.AfterStop(func() error {
        logger.Infoln("service", appName, "stopped")
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
        return errors.New("test")
    }),
)

// register in launcher
ln.ServicesRunner().Register(svc)
```

### Init and register ping pong service

```go
// init
pingPongSvc := service.New(
    service.WithService(pingpong.New(logger, time.Second*5)),
)

// register in launcher
ln.ServicesRunner().Register(pingPongSvc)
```

### Start launcher and all services with graceful shutdown

```go
if err := ln.Run(); err != nil {
    logger.Fatal(err)
}
```
