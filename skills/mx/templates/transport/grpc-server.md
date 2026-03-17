# gRPC Server

Sets up a gRPC transport server using `grpc_transport.NewServer()`. The server implements `types.IService` and supports interceptors, health checks, and reflection.

## GRPCService Interface

Each gRPC service must implement:

```go
type GRPCService interface {
	Name() string
	Register(server *grpc.Server)
}
```

## Example gRPC Service Implementation

```go
package grpcservice

import (
	"google.golang.org/grpc"

	pb "{MODULE_PATH}/gen/{PACKAGE_NAME}/v1"
)

type {SERVICE_STRUCT}GRPCService struct {
	pb.Unimplemented{SERVICE_STRUCT}Server
}

func New{SERVICE_STRUCT}GRPCService() *{SERVICE_STRUCT}GRPCService {
	return &{SERVICE_STRUCT}GRPCService{}
}

func (s *{SERVICE_STRUCT}GRPCService) Name() string { return "{SERVICE_NAME}-grpc" }

func (s *{SERVICE_STRUCT}GRPCService) Register(server *grpc.Server) {
	pb.Register{SERVICE_STRUCT}Server(server, s)
}
```

## Server Setup

```go
import (
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

grpcServer := grpc_transport.NewServer(
	grpc_transport.WithName("grpc-server"),
	grpc_transport.WithLogger(l),  // l is logger.Logger
	grpc_transport.WithConfig(grpc_transport.Config{
		Enabled:            true,
		Addr:               "{GRPC_ADDR}",
		Network:            "tcp",
		ReflectEnabled:     true,
		HealthCheckEnabled: true,
		LoggerEnabled:      true,
		RecoveryEnabled:    true,
	}),
	grpc_transport.WithServices(
		New{SERVICE_STRUCT}GRPCService(),
	),
)

ln.ServicesRunner().Register(
	launcher.NewService(launcher.WithService(grpcServer)),
)
```

## Config Fields

```go
type Config struct {
	Enabled            bool   // default: true
	Addr               string // default: ":9000" (host:port)
	Network            string // default: "tcp" (tcp or udp)
	ReflectEnabled     bool   // enable gRPC reflection
	HealthCheckEnabled bool   // enable grpc_health_v1
	LoggerEnabled      bool   // enable logging interceptor
	RecoveryEnabled    bool   // enable panic recovery interceptor
}
```

## Option Functions

| Option                         | Description                             |
| ------------------------------ | --------------------------------------- |
| `WithConfig(Config)`           | Set full config                         |
| `WithName(string)`             | Server name for logs                    |
| `WithLogger(logger.Logger)`    | Logger instance                         |
| `WithServer(*grpc.Server)`     | Use a custom pre-configured grpc.Server |
| `WithServices(...GRPCService)` | Register gRPC services                  |

## Custom gRPC Server

If you need full control over interceptors:

```go
import "google.golang.org/grpc"

customServer := grpc.NewServer(
	grpc.ChainUnaryInterceptor(myInterceptor),
)

grpcServer := grpc_transport.NewServer(
	grpc_transport.WithServer(customServer),
	grpc_transport.WithServices(myService),
	grpc_transport.WithConfig(grpc_transport.Config{
		Enabled: true,
		Addr:    ":9000",
	}),
)
```

> Note: When providing a custom `grpc.Server` via `WithServer()`, the built-in logging, recovery, and OpenTelemetry interceptors are NOT applied. You must configure them yourself.
