package logger

type Option func(*Options)

type Options struct {
	LogLevel       LogLevel
	LogFormat      LogFormat
	ConsoleColored bool
	AppName        string
	AppVersion     string
	TimeKey        string
}

func WithLogLevel(v LogLevel) Option {
	return func(o *Options) {
		o.LogLevel = v
	}
}

func WithLogFormat(v LogFormat) Option {
	return func(o *Options) {
		o.LogFormat = v
	}
}

func WithConsoleColored(v bool) Option {
	return func(o *Options) {
		o.ConsoleColored = v
	}
}

func WithAppName(v string) Option {
	return func(o *Options) {
		o.AppName = v
	}
}

func WithAppVersion(v string) Option {
	return func(o *Options) {
		o.AppVersion = v
	}
}

func WithTimeKey(v string) Option {
	return func(o *Options) {
		o.TimeKey = v
	}
}
