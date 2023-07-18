package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func initZapLogger(level LogLevel, format LogFormat, consoleColored bool, timeKey string) *zap.Logger {
	atom := zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()

	encoderCfg.TimeKey = "ts"
	if timeKey != "" {
		encoderCfg.TimeKey = timeKey
	}

	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// Default JSON encoder
	encoder := zapcore.NewJSONEncoder(encoderCfg)
	switch format {
	case LoggerFormatConsole:
		if consoleColored {
			encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	logger := zap.New(zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		atom,
	), zap.AddCaller())

	switch level {
	case LogLevelDebug:
		atom.SetLevel(zap.DebugLevel)
	case LogLevelInfo:
		atom.SetLevel(zap.InfoLevel)
	case LogLevelWarning:
		atom.SetLevel(zap.WarnLevel)
	case LogLevelError:
		atom.SetLevel(zap.ErrorLevel)
	case LogLevelFatal:
		atom.SetLevel(zap.FatalLevel)
	case LogLevelPanic:
		atom.SetLevel(zap.PanicLevel)
	default:
		atom.SetLevel(zap.InfoLevel)
	}

	return logger
}

// New - init new logger with options
func New(opts ...Option) Logger {
	return newInternalLogger(opts...)
}

// NewSugared - init new sugared logger with options
func NewExtended(opts ...Option) ExtendedLogger {
	return newInternalLogger(opts...)
}
