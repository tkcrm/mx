package logger

import (
	"go.uber.org/zap"
)

type logger struct {
	*sugaredLogger
}

func newInternalLogger(opts ...Option) *logger {
	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.LogLevel == "" {
		options.LogLevel = LogLevelDebug
	}

	l := initZapLogger(
		options.LogLevel,
		options.LogFormat,
		options.ConsoleColored,
		options.TimeKey,
	)

	if options.AppName != "" {
		l = l.With(
			zap.String("app", options.AppName),
		)
	}

	return &logger{
		sugaredLogger: l.Sugar(),
	}
}

func (l logger) With(args ...any) ExtendedLogger {
	return &logger{l.sugaredLogger.With(args...)}
}

func (l *logger) Sugar() *sugaredLogger {
	return l.sugaredLogger
}
