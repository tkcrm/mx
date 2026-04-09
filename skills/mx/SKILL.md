---
name: mx
description: Guides building Go microservices with the MX framework — service lifecycle, transports (HTTP/gRPC/ConnectRPC), health checks, logging, and ops.
user-invocable: false
---

# MX Microservices Framework

## When to use

- Code imports `github.com/tkcrm/mx` or any of its subpackages
- User asks about creating a Go microservice with MX
- User mentions launcher, services runner, or service lifecycle in the context of MX
- User is setting up gRPC, ConnectRPC, or HTTP transports with MX
- User is configuring health checks, metrics, or profiling via MX ops
- User asks about restart policies, graceful shutdown, or service state management

## How to proceed

1. **Identify the task scope** — determine which MX subsystem is involved:
   - Service creation → see `templates/service/`
   - Launcher bootstrap → see `templates/launcher/`
   - Transport setup (HTTP, gRPC, ConnectRPC) → see `templates/transport/`
   - Client creation → see `templates/clients/`
   - Logger configuration → see `templates/logger/`
   - Ops (health, metrics, profiler) → see `templates/ops/`

2. **Follow framework conventions**:
   - Every service must implement `types.IService` (Name, Start, Stop)
   - Use the functional options pattern (`WithXxx` functions) for all configuration
   - Wrap services with `launcher.NewService(launcher.WithService(svc))` before registering
   - Register services via `launcher.ServicesRunner().Register(...)`

3. **Apply the correct template** from `templates/` and replace placeholders with user-specific values.

4. **Wire everything together** through the Launcher:
   - Create logger → create transports/services → create launcher with ops config → register services → call `launcher.Run()`

5. **Validate the result**:
   - Ensure all services implement `types.IService`
   - Confirm health checkers implement `types.HealthChecker` if needed
   - Check that `launcher.Run()` is the last call in main (it blocks)

## Key principles

- **Interface-first**: Services are defined by `types.IService`. Optional interfaces (`types.HealthChecker`, `types.Enabler`) are detected via duck-typing in `WithService()`.
- **Functional options everywhere**: All MX components use `Option func(*T)` pattern. Never set struct fields directly.
- **Launcher is the orchestrator**: All services (transports, custom services, ops) are registered with and managed by the Launcher's ServicesRunner.
- **Graceful shutdown by default**: The Launcher handles OS signals (SIGTERM, SIGINT, SIGQUIT). First signal triggers graceful shutdown; second signal forces exit.
- **Ops are separate**: Health checks, metrics, and profiler run on a dedicated HTTP server (default port 10000), not on the application transport.
- **Context flows down**: The Launcher creates a root context that is passed to all services. Services should respect `<-ctx.Done()` in their Start function.
- **Restart policies are per-service**: Configure `RestartOnFailure` or `RestartAlways` with exponential backoff on individual services, not globally.
- **Startup priority groups**: Services with `StartupPriority > 0` start in ascending group order (same priority = concurrent within group). All must be ready before the next group. Priority 0 (default) starts last, concurrently.

## Related files

- [architecture.md](architecture.md) — design patterns and lifecycle details
- [stack-overview.md](stack-overview.md) — dependencies and installation
- [project-structure.md](project-structure.md) — directory layout
- [service-checklist.md](service-checklist.md) — step-by-step for adding a service
