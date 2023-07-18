package logger

import (
	"go.uber.org/zap"
)

type LogFormat string

const (
	LoggerFormatConsole LogFormat = "console"
	LoggerFormatJSON    LogFormat = "json"
)

type sugaredLogger = zap.SugaredLogger

type ExtendedLogger interface {
	Logger
	With(...any) ExtendedLogger
	Sugar() *sugaredLogger
}

// Logger common interface
type Logger interface {
	Debug(...any)
	Debugln(...any)
	Debugf(template string, args ...any)
	Debugw(msg string, keysAndValues ...any)

	Info(...any)
	Infoln(...any)
	Infof(template string, args ...any)
	Infow(msg string, keysAndValues ...any)

	Warn(...any)
	Warnln(...any)
	Warnf(template string, args ...any)
	Warnw(msg string, keysAndValues ...any)

	Error(...any)
	Errorln(...any)
	Errorf(template string, args ...any)
	Errorw(msg string, keysAndValues ...any)

	Fatal(...any)
	Fatalln(...any)
	Fatalf(template string, args ...any)
	Fatalw(msg string, keysAndValues ...any)

	Panic(...any)
	Panicln(...any)
	Panicf(template string, args ...any)
	Panicw(msg string, keysAndValues ...any)

	Sync() error
}
