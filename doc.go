// Package mx is a composable Go microservices framework built around a central
// Launcher that orchestrates independent services through a services runner.
//
// MX gives every service a well-defined lifecycle (idle → starting → running →
// stopping → stopped/failed), graceful shutdown on OS signals, health checks,
// metrics, profiling, and ready-made HTTP/gRPC/ConnectRPC transports and clients.
//
// # Getting started
//
//	go get github.com/tkcrm/mx@latest
//
// A minimal application wires a logger, registers services with the launcher,
// and blocks on Run:
//
//	l := logger.NewExtended(logger.WithAppName("app"))
//
//	ln := launcher.New(
//		launcher.WithName("app"),
//		launcher.WithVersion("v1.0.0"),
//		launcher.WithLogger(l),
//	)
//
//	ln.ServicesRunner().Register(
//		launcher.NewService(launcher.WithService(mySvc)),
//	)
//
//	if err := ln.Run(); err != nil { // blocks until shutdown
//		log.Fatal(err)
//	}
//
// # Services
//
// A service is any value implementing the [github.com/tkcrm/mx/launcher/lntypes.IService]
// interface (Name, Start, Stop). Start must block until its context is cancelled
// or its work is done. Wrap a service with [github.com/tkcrm/mx/launcher.NewService]
// and [github.com/tkcrm/mx/launcher.WithService], which duck-types the value for
// the optional lntypes.Enabler and lntypes.HealthChecker interfaces as well.
//
// # Startup priority
//
// Services are started in ascending startup-priority groups. All services in a
// group must reach the running state before the next group starts, while
// services within a group start concurrently. This lets infrastructure such as
// databases and message queues come up first, with the rest of the application
// starting only once they are ready. Priority 0 (the default) starts last, after
// every prioritized group is ready:
//
//	ln.ServicesRunner().Register(
//		launcher.NewService(launcher.WithService(db), launcher.WithStartupPriority(1)),
//		launcher.NewService(launcher.WithService(queue), launcher.WithStartupPriority(1)),
//		launcher.NewService(launcher.WithService(app)), // priority 0 → starts last
//	)
//
// Shutdown order is controlled independently via
// [github.com/tkcrm/mx/launcher.WithRunnerServicesSequence] (None/Fifo/Lifo).
//
// # Restart policies
//
// Per-service restart behaviour is configured with
// [github.com/tkcrm/mx/launcher.WithRestartPolicy]: RestartOnFailure or
// RestartAlways, with a bounded number of retries and exponential backoff.
//
// # Ops
//
// When ops are enabled via [github.com/tkcrm/mx/launcher.WithOpsConfig], the
// launcher runs a dedicated HTTP server (default port 10000) exposing a liveness
// probe (/livez), a readiness probe (/readyz), a legacy health endpoint
// (/healthy), Prometheus metrics (/metrics), and the pprof profiler
// (/debug/pprof).
//
// # Subpackages
//
//   - launcher — service orchestration, lifecycle, restart policies, ops wiring.
//   - launcher/lntypes — core interfaces (IService, HealthChecker, Enabler,
//     StateProvider) and ServiceState.
//   - launcher/ops — health, metrics, and profiler operational services.
//   - logger — structured logging backed by go.uber.org/zap.
//   - transport/http_transport — net/http server as a managed service.
//   - transport/grpc_transport — gRPC server with interceptors, health, reflection.
//   - transport/connectrpc_transport — ConnectRPC (gRPC-compatible) server.
//   - clients/grpc_client, clients/connectrpc_client — generic client factories.
//   - util — assorted helpers (JSON, structs, files, timing).
//
// All MX components follow the functional-options pattern (WithXxx). See the
// package-level documentation of each subpackage for details.
package mx
