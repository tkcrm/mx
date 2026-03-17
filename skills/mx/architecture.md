# Architecture

## Overview

MX is a composable Go microservices framework built around a central **Launcher** that orchestrates independent **Services** through a **ServicesRunner**. Each service has a well-defined lifecycle managed by the framework.

## Core Interfaces

### IService (required)

Every service must implement this interface from `launcher/types`:

```go
type IService interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}
```

### HealthChecker (optional)

Services can optionally implement health checking:

```go
type HealthChecker interface {
    Name() string
    Interval() time.Duration
    Healthy(ctx context.Context) error
}
```

### Enabler (optional)

Services can be conditionally enabled/disabled:

```go
type Enabler interface {
    Enabled() bool
}
```

### StateProvider (internal)

Exposes current lifecycle state — used by liveness/readiness probes:

```go
type StateProvider interface {
    Name() string
    State() ServiceState
}
```

## Service Lifecycle State Machine

```text
idle → starting → running → stopping → stopped
                     ↓                     ↑
                  failed ─────────────────┘
                     ↑         (restart policy)
                     └── starting (retry)
```

States: `ServiceStateIdle`, `ServiceStateStarting`, `ServiceStateRunning`, `ServiceStateStopping`, `ServiceStateStopped`, `ServiceStateFailed`

## Launcher Architecture

```text
Launcher
├── Options (name, version, hooks, ops config, shutdown timeout)
├── ServicesRunner
│   ├── Service (wraps IService + state + options)
│   ├── Service ...
│   └── Service ...
└── Context (root context, cancelled on shutdown)
```

### Launcher Lifecycle

1. Register ops services (if `OpsConfig.Enabled`)
2. Run `BeforeStart` hooks sequentially
3. Start all services in parallel goroutines
4. Run `AfterStart` hooks sequentially
5. Wait for: service error, OS signal, or context cancellation
6. On first signal: cancel context, log graceful shutdown message
7. On second signal: force exit (`os.Exit(1)`)
8. Wait for all service goroutines to finish
9. Run `BeforeStop` hooks
10. Stop services according to `RunnerServicesSequence`
11. Run `AfterStop` hooks

### Shutdown Sequences

- `RunnerServicesSequenceNone` (default) — stop all services in parallel
- `RunnerServicesSequenceFifo` — stop in registration order
- `RunnerServicesSequenceLifo` — stop in reverse registration order

## Service Wrapper

`launcher.NewService()` wraps any value into a managed `Service` with:

- Lifecycle hooks: `BeforeStart`, `AfterStart`, `AfterStartFinished`, `BeforeStop`, `AfterStop`
- `ShutdownTimeout` (default 10s) — max time for Stop to complete
- `StartupTimeout` — max time for Start to signal (0 = no timeout)
- `RestartPolicy` — automatic restart on failure or always

### Restart Policies

```go
RestartNever     // default — no automatic restarts
RestartOnFailure // restart only when Start returns an error
RestartAlways    // restart on any exit (error or clean)
```

`RestartPolicy` supports:

- `MaxRetries` — max restart attempts (0 = unlimited)
- `Delay` — initial backoff delay
- `MaxDelay` — cap for exponential backoff

## Duck-Typing via WithService()

`launcher.WithService(svc any)` introspects the passed value for:

- `Name() string` → sets service name
- `Start(ctx context.Context) error` → sets start function
- `Stop(ctx context.Context) error` → sets stop function
- `Enabled() bool` (types.Enabler) → sets enabled state
- `HealthChecker` interface → registers health checker

This allows wrapping any struct without requiring explicit interface satisfaction at compile time.

## Functional Options Pattern

All MX components use the same pattern:

```go
type Option func(*T)

func WithXxx(v V) Option {
    return func(t *T) { t.xxx = v }
}
```

Used in: `launcher.New()`, `launcher.NewService()`, `logger.New()`, `http_transport.NewServer()`, `grpc_transport.NewServer()`, `connectrpc_transport.NewServer()`, and all client factories.

## Ops Services

When `OpsConfig.Enabled` is true, the launcher automatically creates HTTP server(s) for operational concerns:

### Health Endpoints

- `/healthy` — legacy health check (HealthChecker results). Returns 200/424/503.
- `/livez` — liveness probe. Returns 200 if no service is in `Failed` state; 503 otherwise.
- `/readyz` — readiness probe. Combines service state + health check results. Returns 200 only when all services are `Running` and all health checks pass.

### Metrics

- `/metrics` — Prometheus metrics endpoint

### Profiler

- `/debug/pprof` — Go pprof profiler endpoints

All ops services share an HTTP server on the configured port (default 10000). Different services can use different ports if configured.

## Transport Layer

MX provides three transport implementations, all implementing `IService`:

### HTTP Transport (`http_transport`)

- Standard `net/http` server with configurable timeouts
- Tracing middleware (OpenTelemetry)
- Basic auth middleware

### gRPC Transport (`grpc_transport`)

- Full gRPC server with interceptor chains
- Logging interceptor, recovery interceptor, OpenTelemetry stats handler
- gRPC health check service and reflection support
- Services registered via `GRPCService` interface: `Name()` + `Register(*grpc.Server)`

### ConnectRPC Transport (`connectrpc_transport`)

- HTTP-based RPC compatible with gRPC
- Services registered via `ConnectRPCService` interface: `Name()` + `RegisterHandler(...connect.HandlerOption) (string, http.Handler)`
- gRPC reflection support via `grpcreflect.Reflector`
- Custom handler wrapper support

## Client Factories

Both client factories use Go generics for type-safe client creation:

### gRPC Client

```go
client, err := grpc_client.New[T](config, logger, initFn, opts...)
```

### ConnectRPC Client

```go
client, err := connectrpc_client.New[T](config, logger, initFn, opts...)
```

## Logger

Two logger interfaces:

- `Logger` — standard logging (Debug/Info/Warn/Error/Fatal/Panic with f/ln/w variants)
- `ExtendedLogger` — adds `Sync()`, `Std()`, `Sugar()` methods

Backed by `go.uber.org/zap`. Supports JSON and console formats, configurable levels, caller info, and stack traces.
