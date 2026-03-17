# Stack Overview

## Language and Runtime

- **Go 1.26+** (uses generics in client factories)
- Module path: `github.com/tkcrm/mx`

## Installation

```bash
go get github.com/tkcrm/mx@latest
```

Individual subpackages can be imported as needed:

```go
import (
    "github.com/tkcrm/mx/launcher"
    "github.com/tkcrm/mx/launcher/types"
    "github.com/tkcrm/mx/logger"
    "github.com/tkcrm/mx/transport/http_transport"
    "github.com/tkcrm/mx/transport/grpc_transport"
    "github.com/tkcrm/mx/transport/connectrpc_transport"
    "github.com/tkcrm/mx/clients/grpc_client"
    "github.com/tkcrm/mx/clients/connectrpc_client"
    "github.com/tkcrm/mx/launcher/ops"
)
```

## Core Dependencies

| Dependency | Purpose |
| --- | --- |
| `go.uber.org/zap` | Structured logging (Logger and ExtendedLogger) |
| `google.golang.org/grpc` | gRPC server and client |
| `connectrpc.com/connect` | ConnectRPC server and client |
| `connectrpc.com/grpcreflect` | gRPC reflection for ConnectRPC |
| `go.opentelemetry.io/contrib/instrumentation` | Distributed tracing (gRPC, HTTP) |
| `github.com/prometheus/client_golang` | Prometheus metrics export |
| `golang.org/x/sync` | errgroup for concurrent shutdown |
| `github.com/goccy/go-json` | High-performance JSON encoding |
| `google.golang.org/protobuf` | Protocol Buffers support |

## gRPC Middleware (via `go-grpc-middleware/v2`)

- `interceptors/logging` — structured request/response logging
- `interceptors/recovery` — panic recovery with stack traces

## Build Tools

The project Makefile provides:

- `make fmt` — run `gofumpt` formatter
- `make lint` — run `golangci-lint`
