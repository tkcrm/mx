# HTTP Server

Sets up an HTTP transport server using `http_transport.NewServer()`. The server implements `types.IService` and can be registered with the launcher.

## Basic Setup

```go
import (
	"net/http"

	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/http_transport"
)

mux := http.NewServeMux()
mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

httpServer := http_transport.NewServer(
	http_transport.WithName("{SERVICE_NAME}"),
	http_transport.WithLogger(l),  // l is logger.ExtendedLogger
	http_transport.WithHandler(mux),
	http_transport.WithConfig(http_transport.Config{
		Enabled: true,
		Address: "{HTTP_ADDR}",
		Network: "tcp",
	}),
)

ln.ServicesRunner().Register(
	launcher.NewService(launcher.WithService(httpServer)),
)
```

## Config Fields

```go
type Config struct {
	Enabled           bool   // default: false
	Address           string // default: ":8080" (host:port)
	Network           string // default: "tcp" (tcp or udp)
	NoTrace           bool   // disable OpenTelemetry tracing middleware
	ReadTimeout       int    // seconds, default: 5
	WriteTimeout      int    // seconds, default: 10
	IdleTimeout       int    // seconds, default: 60
	ReadHeaderTimeout int    // seconds, default: 10
}
```

## Option Functions

| Option                              | Description                      |
| ----------------------------------- | -------------------------------- |
| `WithConfig(Config)`                | Set full config                  |
| `WithName(string)`                  | Server name for logs             |
| `WithLogger(logger.ExtendedLogger)` | Logger instance                  |
| `WithHandler(http.Handler)`         | HTTP handler (mux, router, etc.) |
| `WithReadTimeout(int)`              | Read timeout in seconds          |
| `WithWriteTimeout(int)`             | Write timeout in seconds         |
| `WithIdleTimeout(int)`              | Idle timeout in seconds          |
| `WithReadHeaderTimeout(int)`        | Read header timeout in seconds   |

## Custom Timeouts

```go
httpServer := http_transport.NewServer(
	http_transport.WithName("api-server"),
	http_transport.WithLogger(l),
	http_transport.WithHandler(mux),
	http_transport.WithConfig(http_transport.Config{
		Enabled: true,
		Address: ":8080",
		Network: "tcp",
	}),
	http_transport.WithReadTimeout(10),
	http_transport.WithWriteTimeout(30),
	http_transport.WithIdleTimeout(120),
)
```

> Note: The HTTP server automatically applies `TracingMiddleware` unless `NoTrace` is set to `true` in the config.
