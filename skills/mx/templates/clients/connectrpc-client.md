# ConnectRPC Client

Type-safe ConnectRPC client factory using Go generics. Returns a fully initialized client of any Connect-generated type.

## Basic Usage

```go
import (
	"github.com/tkcrm/mx/clients/connectrpc_client"

	pbconnect "{MODULE_PATH}/gen/{PACKAGE_NAME}/v1/{PACKAGE_NAME}v1connect"
)

client, err := connectrpc_client.New(
	connectrpc_client.Config{
		Name: "{SERVICE_NAME}-client",
		Addr: "http://localhost:{CONNECTRPC_ADDR}",
	},
	l,  // logger (can be nil)
	pbconnect.New{SERVICE_STRUCT}Client,  // connect-generated client constructor
)
if err != nil {
	log.Fatal(err)
}
```

## Config Fields

```go
type Config struct {
	Name string // required — client name for logs
	Addr string // required — server base URL (e.g., "http://localhost:9000")
}
```

## Option Functions

| Option | Description |
| --- | --- |
| `WithName(string)` | Override client name |
| `WithContext(context.Context)` | Set context |
| `WithHttpClient(connect.HTTPClient)` | Custom HTTP client (default: `http.DefaultClient`) |
| `WithConnectrpcOpts(...connect.ClientOption)` | Additional Connect client options |

## With Custom HTTP Client

```go
import "net/http"

httpClient := &http.Client{
	Timeout: 30 * time.Second,
}

client, err := connectrpc_client.New(
	connectrpc_client.Config{
		Name: "{SERVICE_NAME}-client",
		Addr: "http://localhost:9000",
	},
	l,
	pbconnect.New{SERVICE_STRUCT}Client,
	connectrpc_client.WithHttpClient(httpClient),
)
```

## With Connect Options

```go
import "connectrpc.com/connect"

client, err := connectrpc_client.New(
	connectrpc_client.Config{
		Name: "{SERVICE_NAME}-client",
		Addr: "http://localhost:9000",
	},
	l,
	pbconnect.New{SERVICE_STRUCT}Client,
	connectrpc_client.WithConnectrpcOpts(
		connect.WithGRPC(),  // use gRPC protocol instead of Connect
		connect.WithInterceptors(myInterceptor),
	),
)
```

> Note: The `fn` parameter (third argument) is the Connect-generated `NewXxxClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) XxxClient` function. The generic type `T` is inferred from it.
