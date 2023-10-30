package grpc_transport

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/tkcrm/mx/logger"
)

func InterceptorLogger(l logger.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		largs := append([]any{"msg", msg}, fields...)
		switch lvl {
		case logging.LevelDebug:
			l.Debug(largs...)
		case logging.LevelInfo:
			l.Info(largs...)
		case logging.LevelWarn:
			l.Warn(largs...)
		case logging.LevelError:
			l.Error(largs...)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
