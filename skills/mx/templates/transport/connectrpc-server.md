# ConnectRPC Server

Sets up a ConnectRPC transport server using `connectrpc_transport.NewServer()`. ConnectRPC is an HTTP-based RPC framework compatible with gRPC. The server implements `types.IService`.

## ConnectRPCService Interface

Each ConnectRPC service must implement:

```go
type ConnectRPCService interface {
	Name() string
	RegisterHandler(opts ...connect.HandlerOption) (string, http.Handler)
}
```

## Example ConnectRPC Service Implementation

```go
package connectrpcservice

import (
	"net/http"

	"connectrpc.com/connect"

	pb "{MODULE_PATH}/gen/{PACKAGE_NAME}/v1"
	pbconnect "{MODULE_PATH}/gen/{PACKAGE_NAME}/v1/{PACKAGE_NAME}v1connect"
)

type {SERVICE_STRUCT}ConnectService struct{}

func New{SERVICE_STRUCT}ConnectService() *{SERVICE_STRUCT}ConnectService {
	return &{SERVICE_STRUCT}ConnectService{}
}

func (s *{SERVICE_STRUCT}ConnectService) Name() string { return "{SERVICE_NAME}-connectrpc" }

func (s *{SERVICE_STRUCT}ConnectService) RegisterHandler(
	opts ...connect.HandlerOption,
) (string, http.Handler) {
	return pbconnect.New{SERVICE_STRUCT}Handler(s, opts...)
}
```

## Server Setup

```go
import (
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/connectrpc_transport"
)

connectServer := connectrpc_transport.NewServer(
	connectrpc_transport.WithName("connectrpc-server"),
	connectrpc_transport.WithLogger(l),  // l is logger.Logger
	connectrpc_transport.WithConfig(connectrpc_transport.Config{
		Enabled: true,
		Addr:    "{CONNECTRPC_ADDR}",
	}),
	connectrpc_transport.WithServices(
		New{SERVICE_STRUCT}ConnectService(),
	),
	connectrpc_transport.WithReflection(
		"package.v1.{SERVICE_STRUCT}",  // fully qualified protobuf service names
	),
)

ln.ServicesRunner().Register(
	launcher.NewService(launcher.WithService(connectServer)),
)
```

## Config Fields

```go
type Config struct {
	Enabled        bool   // default: true
	Addr           string // default: ":9000" (host:port)
	ReflectEnabled bool   // enable gRPC reflection
}
```

## Option Functions

| Option                                                      | Description                             |
| ----------------------------------------------------------- | --------------------------------------- |
| `WithConfig(Config)`                                        | Set full config                         |
| `WithName(string)`                                          | Server name for logs                    |
| `WithLogger(logger.Logger)`                                 | Logger instance                         |
| `WithServeMux(*http.ServeMux)`                              | Use a custom ServeMux                   |
| `WithHttpServer(*http.Server)`                              | Use a custom http.Server                |
| `WithReflection(...string)`                                 | Enable reflection with service names    |
| `WithServerHandlerWrapper(func(http.Handler) http.Handler)` | Wrap the final handler (middleware)     |
| `WithServices(...ConnectRPCService)`                        | Register ConnectRPC services            |
| `WithConnectRPCOptions(...connect.HandlerOption)`           | Global handler options for all services |

## Adding Middleware

Use `WithServerHandlerWrapper` to wrap the HTTP handler with middleware (CORS, auth, etc.):

```go
connectServer := connectrpc_transport.NewServer(
	connectrpc_transport.WithServices(myService),
	connectrpc_transport.WithServerHandlerWrapper(func(h http.Handler) http.Handler {
		return corsMiddleware(h)
	}),
	connectrpc_transport.WithConfig(connectrpc_transport.Config{
		Enabled: true,
		Addr:    ":9000",
	}),
)
```

## Adding Connect Handler Options

Apply options globally to all registered services:

```go
connectrpc_transport.WithConnectRPCOptions(
	connect.WithInterceptors(myInterceptor),
)
```
