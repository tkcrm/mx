# Launcher Setup

Complete launcher bootstrap for an MX application. This is the main entry point in `cmd/{APP_NAME}/main.go`.

## Minimal Setup

```go
package main

import (
	"log"

	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
)

func main() {
	l := logger.NewExtended(
		logger.WithAppName("{APP_NAME}"),
		logger.WithLogLevel(logger.LogLevelInfo),
	)

	ln := launcher.New(
		launcher.WithName("{APP_NAME}"),
		launcher.WithVersion("{APP_VERSION}"),
		launcher.WithLogger(l),
	)

	// Register services here
	// ln.ServicesRunner().Register(...)

	if err := ln.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Full Setup with Ops

```go
package main

import (
	"log"
	"time"

	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/launcher/ops"
	"github.com/tkcrm/mx/logger"
)

func main() {
	l := logger.NewExtended(
		logger.WithAppName("{APP_NAME}"),
		logger.WithAppVersion("{APP_VERSION}"),
		logger.WithLogLevel(logger.LogLevelInfo),
		logger.WithLogFormat(logger.LoggerFormatJSON),
	)

	ln := launcher.New(
		launcher.WithName("{APP_NAME}"),
		launcher.WithVersion("{APP_VERSION}"),
		launcher.WithLogger(l),
		launcher.WithAppStartStopLog(true),
		launcher.WithGlobalShutdownTimeout(30*time.Second),
		launcher.WithRunnerServicesSequence(launcher.RunnerServicesSequenceLifo),
		launcher.WithOpsConfig(ops.Config{
			Enabled: true,
			Network: "tcp",
			Healthy: ops.HealthCheckerConfig{
				Enabled:       true,
				Path:          "/healthy",
				Port:          "{OPS_PORT}",
				LivenessPath:  "/livez",
				ReadinessPath: "/readyz",
			},
			Metrics: ops.MetricsConfig{
				Enabled: true,
				Port:    "{OPS_PORT}",
			},
			Profiler: ops.ProfilerConfig{
				Enabled: true,
				Port:    "{OPS_PORT}",
			},
		}),
		launcher.WithBeforeStart(func() error {
			l.Info("initializing application...")
			return nil
		}),
		launcher.WithAfterStop(func() error {
			return l.Sync()
		}),
	)

	// Register your services
	// ln.ServicesRunner().Register(
	//     launcher.NewService(launcher.WithService(myService)),
	//     launcher.NewService(launcher.WithService(grpcServer)),
	// )

	if err := ln.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Launcher Options Reference

| Option                                     | Description                                           |
| ------------------------------------------ | ----------------------------------------------------- |
| `WithName(string)`                         | Application name                                      |
| `WithVersion(string)`                      | Application version                                   |
| `WithLogger(logger.ExtendedLogger)`        | Logger instance                                       |
| `WithContext(context.Context)`             | Custom root context (default: `context.Background()`) |
| `WithSignal(bool)`                         | Enable OS signal handling (default: `true`)           |
| `WithAppStartStopLog(bool)`                | Log app started/stopped messages                      |
| `WithGlobalShutdownTimeout(time.Duration)` | Max total shutdown time (0 = no limit)                |
| `WithRunnerServicesSequence(seq)`          | Shutdown order: None/Fifo/Lifo                        |
| `WithOpsConfig(ops.Config)`                | Ops server configuration                              |
| `WithBeforeStart(func() error)`            | Hook before services start                            |
| `WithAfterStart(func() error)`             | Hook after services start                             |
| `WithBeforeStop(func() error)`             | Hook before services stop                             |
| `WithAfterStop(func() error)`              | Hook after services stop                              |

## Programmatic Stop

You can stop the launcher from code:

```go
ln.Stop() // cancels the root context, triggering graceful shutdown
```

## Adding Hooks After Creation

```go
ln.AddBeforeStartHooks(func() error { /* ... */ return nil })
ln.AddAfterStopHooks(func() error { /* ... */ return nil })
```
