package logger

import (
	"go.uber.org/zap/zapcore"
)

type LogLevel string

func (l LogLevel) String() string { return string(l) }

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
	LogLevelPanic LogLevel = "panic"
)

// GetAllLevels return all log levels. Used in validation.
var allLevels = []any{
	LogLevelDebug.String(),
	LogLevelInfo.String(),
	LogLevelWarn.String(),
	LogLevelError.String(),
	LogLevelFatal.String(),
	LogLevelPanic.String(),
}

// safeLevel converts string representation into log level.
func safeLevel(level LogLevel) zapcore.Level {
	switch level {
	default:
		return zapcore.InfoLevel
	case LogLevelDebug:
		return zapcore.DebugLevel
	case LogLevelWarn:
		return zapcore.WarnLevel
	case LogLevelError:
		return zapcore.ErrorLevel
	case LogLevelPanic:
		return zapcore.PanicLevel
	case LogLevelFatal:
		return zapcore.FatalLevel
	}
}
