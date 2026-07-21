# Project Structure

## MX Framework Layout

```text
mx/
├── launcher/                          # Service orchestration
│   ├── launcher.go                    # ILauncher implementation (New, Run, Stop)
│   ├── service.go                     # Service wrapper with state machine
│   ├── service_options.go             # ServiceOption functions
│   ├── services_runner.go             # IServicesRunner (Register, Get, Services)
│   ├── options.go                     # Launcher Option functions
│   ├── restart_policy.go              # RestartMode, RestartPolicy
│   ├── signal.go                      # OS signal set (SIGTERM, SIGINT, SIGQUIT)
│   ├── lntypes/                       # Core interfaces
│   │   └── types.go                   # IService, HealthChecker, Enabler, StateProvider, ServiceState
│   ├── ops/                           # Operational services
│   │   ├── ops.go                     # Ops factory (New)
│   │   ├── config.go                  # Config (top-level ops config)
│   │   ├── health.go                  # HealthCheckerConfig + /healthy, /livez, /readyz handlers
│   │   ├── metrics.go                 # MetricsConfig + Prometheus /metrics handler
│   │   ├── profiler.go                # ProfilerConfig + pprof /debug/pprof handler
│   │   └── sentry/                    # Sentry error tracking integration
│   └── services/
│       └── pingpong/                  # Example ping-pong service
│           └── ping_pong.go
├── logger/                            # Structured logging
│   ├── logger.go                      # New, NewExtended, With, WithExtended
│   ├── interface.go                   # Logger, ExtendedLogger interfaces
│   ├── config.go                      # Config struct
│   ├── options.go                     # Option functions
│   ├── level.go                       # LogLevel constants
│   └── format.go                      # LogFormat constants (json, console)
├── transport/                         # Network transport layers
│   ├── http_transport/
│   │   ├── http.go                    # HTTPServer (NewServer, Start, Stop)
│   │   ├── config.go                  # Config (address, timeouts)
│   │   ├── options.go                 # Option functions
│   │   ├── tracing.go                 # TracingMiddleware
│   │   └── basicauth.go              # Basic auth middleware
│   ├── grpc_transport/
│   │   ├── grpc.go                    # GRPCServer (NewServer, Start, Stop)
│   │   ├── config.go                  # Config (addr, reflection, health, recovery)
│   │   ├── options.go                 # Option functions, GRPCService interface
│   │   ├── recovery.go               # RecoveryFunc
│   │   ├── reflection.go             # Reflection service
│   │   └── logger.go                 # InterceptorLogger
│   └── connectrpc_transport/
│       ├── connectrpc.go              # ConnectRPCServer (NewServer, Start, Stop)
│       ├── config.go                  # Config (addr, reflection)
│       └── options.go                 # Option functions, ConnectRPCService interface
├── clients/                           # Client factories
│   ├── grpc_client/
│   │   ├── client.go                  # New[T] generic factory
│   │   └── config.go                  # Config, Option
│   └── connectrpc_client/
│       ├── client.go                  # New[T] generic factory
│       └── config.go                  # Config, Option
└── util/                              # Utilities
    ├── structs/                       # Struct utilities (lookup)
    ├── files/                         # File utilities
    ├── json.go                        # JSON helpers
    └── timeit.go                      # Execution timing helper
```

## Typical Application Layout Using MX

```text
{APP_NAME}/
├── cmd/
│   └── {APP_NAME}/
│       └── main.go                    # Launcher bootstrap, service registration
├── internal/
│   ├── config/
│   │   └── config.go                  # App config (embeds transport/ops configs)
│   ├── services/
│   │   └── {SERVICE_NAME}/
│   │       ├── service.go             # IService implementation
│   │       └── health.go              # HealthChecker (optional)
│   └── transport/
│       ├── grpc/
│       │   └── service.go             # GRPCService implementation (Register)
│       └── connectrpc/
│           └── service.go             # ConnectRPCService implementation (RegisterHandler)
├── proto/                             # Protobuf definitions (if using gRPC/ConnectRPC)
│   └── {SERVICE_NAME}/
│       └── v1/
│           └── {SERVICE_NAME}.proto
├── go.mod
├── go.sum
└── Makefile
```
