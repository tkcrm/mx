package logger_test

import (
	"testing"

	"github.com/tkcrm/micro/logger"
)

func TestLogger(t *testing.T) {
	l := logger.New(
		logger.WithAppName("test"),
		logger.WithLogLevel(logger.LogLevelDebug),
	)

	l.Info("Hello world")
}

func Test_LoggerWith(t *testing.T) {
	l := logger.NewExtended(
		logger.WithAppName("test"),
		logger.WithLogLevel(logger.LogLevelDebug),
		logger.WithLogFormat(logger.LoggerFormatConsole),
	).With("key", "value").With("key2", "value2")

	l = l.With("key3", "value3")

	l.Infof("some test value: %d", 1234)
}
