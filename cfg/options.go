package cfg

type Option func(*options)

type options struct {
	envFile string
	envPath string
}

func WithEnvFile(v string) Option {
	return func(o *options) {
		o.envFile = v
	}
}

func WithEnvPath(v string) Option {
	return func(o *options) {
		o.envPath = v
	}
}
