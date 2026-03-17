# gRPC Client

Type-safe gRPC client factory using Go generics. Returns a fully initialized client of any protobuf-generated type.

## Basic Usage

```go
import (
	"github.com/tkcrm/mx/clients/grpc_client"

	pb "{MODULE_PATH}/gen/{PACKAGE_NAME}/v1"
)

client, err := grpc_client.New(
	grpc_client.Config{
		Name:     "{SERVICE_NAME}-client",
		Addr:     "{GRPC_ADDR}",
		Insecure: true,
	},
	l,  // logger (can be nil)
	pb.New{SERVICE_STRUCT}Client,  // protobuf-generated client constructor
)
if err != nil {
	log.Fatal(err)
}
```

## Config Fields

```go
type Config struct {
	Name     string // required — client name for logs
	Addr     string // required — server address (host:port)
	UseTls   bool   // use TLS credentials
	Insecure bool   // use insecure credentials (for development)
}
```

## Option Functions

| Option                            | Description                  |
| --------------------------------- | ---------------------------- |
| `WithName(string)`                | Override client name         |
| `WithContext(context.Context)`    | Set context                  |
| `WithGrpcOpt(...grpc.DialOption)` | Additional gRPC dial options |

## With TLS

```go
client, err := grpc_client.New(
	grpc_client.Config{
		Name:   "{SERVICE_NAME}-client",
		Addr:   "api.example.com:443",
		UseTls: true,
	},
	l,
	pb.New{SERVICE_STRUCT}Client,
)
```

## With Custom Dial Options

```go
client, err := grpc_client.New(
	grpc_client.Config{
		Name:     "{SERVICE_NAME}-client",
		Addr:     "localhost:9000",
		Insecure: true,
	},
	l,
	pb.New{SERVICE_STRUCT}Client,
	grpc_client.WithGrpcOpt(
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(10*1024*1024)),
	),
)
```

> Note: The `fn` parameter (third argument) is the protobuf-generated `NewXxxClient(cc grpc.ClientConnInterface) XxxClient` function. The generic type `T` is inferred from it.
