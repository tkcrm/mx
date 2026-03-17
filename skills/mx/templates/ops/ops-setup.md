# Ops Setup

Ops services provide health checks, Prometheus metrics, and pprof profiling on a dedicated HTTP server. They are automatically created and registered by the Launcher when `OpsConfig.Enabled` is true.

## Configuration

Pass `ops.Config` to the launcher via `launcher.WithOpsConfig()`:

```go
import (
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/launcher/ops"
)

ln := launcher.New(
	launcher.WithLogger(l),
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
)
```

## Config Structure

```go
type Config struct {
	Enabled        bool   // master switch for all ops services
	Network        string // "tcp" or "udp" (default: "tcp")
	TracingEnabled bool   // enable OpenTelemetry tracing on ops HTTP server
	Metrics        MetricsConfig
	Healthy        HealthCheckerConfig
	Profiler       ProfilerConfig
}

type HealthCheckerConfig struct {
	Enabled       bool   // enable health endpoints
	Path          string // legacy health path (default: "/healthy")
	Port          string // HTTP port (default: "10000")
	LivenessPath  string // liveness probe path (default: "/livez")
	ReadinessPath string // readiness probe path (default: "/readyz")
}

type MetricsConfig struct {
	Enabled bool   // enable Prometheus /metrics
	Port    string // HTTP port (default: "10000")
}

type ProfilerConfig struct {
	Enabled bool   // enable pprof /debug/pprof
	Port    string // HTTP port (default: "10000")
}
```

## Endpoints

When all ops services share the same port (e.g., `10000`), a single HTTP server is created:

| Path           | Purpose                                     | Status Codes  |
| -------------- | ------------------------------------------- | ------------- |
| `/healthy`     | Legacy health check (HealthChecker results) | 200, 424, 503 |
| `/livez`       | Liveness probe (service state only)         | 200, 503      |
| `/readyz`      | Readiness probe (state + health checks)     | 200, 424, 503 |
| `/metrics`     | Prometheus metrics                          | 200           |
| `/debug/pprof` | Go profiler                                 | 200           |

## How Health Checking Works

1. Services implementing `types.HealthChecker` are auto-detected when wrapped with `launcher.WithService()`
2. The ops health checker runs each service's `Healthy()` method on its configured `Interval()`
3. Results are stored and served on the health endpoints

### Response Codes

| Code | `/healthy` meaning     | `/livez` meaning             | `/readyz` meaning                 |
| ---- | ---------------------- | ---------------------------- | --------------------------------- |
| 200  | All checks pass        | No service failed            | All running + all checks pass     |
| 424  | A service is starting  | —                            | A service is starting or idle     |
| 503  | A check returned error | A service is in Failed state | A service failed or check errored |

## Different Ports

Services can run on different ports. When different ports are specified, separate HTTP servers are created automatically:

```go
Healthy: ops.HealthCheckerConfig{
	Enabled: true,
	Port:    "10000",
},
Metrics: ops.MetricsConfig{
	Enabled: true,
	Port:    "10001",  // separate server for metrics
},
```

## Kubernetes Probe Configuration

```yaml
livenessProbe:
  httpGet:
    path: /livez
    port: { OPS_PORT }
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: { OPS_PORT }
  initialDelaySeconds: 5
  periodSeconds: 5
```
