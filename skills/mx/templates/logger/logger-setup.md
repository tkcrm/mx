# Logger Setup

MX provides a structured logger backed by `go.uber.org/zap`. Two interfaces are available:

- `Logger` — standard logging methods (Debug/Info/Warn/Error/Fatal/Panic with f/ln/w variants)
- `ExtendedLogger` — adds `Sync()`, `Std()`, and `Sugar()` methods

The launcher requires `ExtendedLogger`; transports accept either.

## Basic Logger

```go
import "github.com/tkcrm/mx/logger"

l := logger.New(
	logger.WithAppName("{APP_NAME}"),
	logger.WithLogLevel(logger.LogLevelInfo),
)
```

## Extended Logger (for Launcher)

```go
l := logger.NewExtended(
	logger.WithAppName("{APP_NAME}"),
	logger.WithAppVersion("{APP_VERSION}"),
	logger.WithLogLevel(logger.LogLevelInfo),
	logger.WithLogFormat(logger.LoggerFormatJSON),
)

// Don't forget to sync on shutdown
defer l.Sync()
```

## Full Configuration

```go
l := logger.NewExtended(
	logger.WithConfig(logger.Config{
		Format:         logger.LoggerFormatJSON,    // "json" or "console"
		Level:          logger.LogLevelDebug,        // "debug", "info", "warn", "error", "fatal", "panic"
		ConsoleColored: false,                       // colored output (console format only)
		Trace:          logger.LogLevelFatal,        // stack trace level
		WithCaller:     true,                        // show caller file:line
		WithStackTrace: false,                       // show stack traces
	}),
	logger.WithAppName("{APP_NAME}"),
	logger.WithAppVersion("{APP_VERSION}"),
)
```

## Option Functions

| Option                      | Description                            |
| --------------------------- | -------------------------------------- |
| `WithConfig(Config)`        | Set full config                        |
| `WithAppName(string)`       | Add `app` field to all log entries     |
| `WithAppVersion(string)`    | Add `version` field to all log entries |
| `WithLogLevel(LogLevel)`    | Set minimum log level                  |
| `WithLogFormat(LogFormat)`  | Set output format (json/console)       |
| `WithConsoleColored(bool)`  | Enable colored console output          |
| `WithCaller(bool)`          | Show caller information                |
| `WithStackTrace(bool)`      | Enable stack traces                    |
| `WithTimeKey(string)`       | Custom time field key                  |
| `WithZapOption(zap.Option)` | Pass raw zap options                   |

## Log Levels

| Constant        | Value     |
| --------------- | --------- |
| `LogLevelDebug` | `"debug"` |
| `LogLevelInfo`  | `"info"`  |
| `LogLevelWarn`  | `"warn"`  |
| `LogLevelError` | `"error"` |
| `LogLevelFatal` | `"fatal"` |
| `LogLevelPanic` | `"panic"` |

## Log Formats

| Constant              | Value              |
| --------------------- | ------------------ |
| `LoggerFormatJSON`    | `"json"` (default) |
| `LoggerFormatConsole` | `"console"`        |

## Adding Context to Logger

Use `logger.With` or `logger.WithExtended` to create a child logger with additional fields:

```go
svcLogger := logger.With(l, "service", "my-service", "port", 9000)
svcLogger.Info("starting")
// Output: {"level":"info","ts":"...","msg":"starting","service":"my-service","port":9000}
```

## Default Loggers

For quick prototyping:

```go
l := logger.Default()         // returns Logger (JSON, debug level)
l := logger.DefaultExtended() // returns ExtendedLogger (JSON, debug level)
```
