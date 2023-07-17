package logger

type LogLevel string

func (l LogLevel) String() string {
	return string(l)
}

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
	LogLevelPanic   LogLevel = "panic"
)

// GetAllLevels return all log levels. Used in validation.
func GetAllLevels() []any {
	return []any{
		LogLevelDebug.String(),
		LogLevelInfo.String(),
		LogLevelWarning.String(),
		LogLevelError.String(),
		LogLevelFatal.String(),
		LogLevelPanic.String(),
	}
}
