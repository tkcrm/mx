package logger

import (
	"go.uber.org/zap"
)

type logger struct {
	*SugaredLogger
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
		SugaredLogger: l.Sugar(),
	}
}

func (l logger) With(args ...any) Logger {
	return &logger{l.SugaredLogger.With(args...)}
}

func (l *logger) Sugar() *SugaredLogger {
	return l.SugaredLogger
}
