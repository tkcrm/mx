package pingpong

import "time"

type Option func(*PingPong)

func WithTimeout(v time.Duration) Option {
	return func(pp *PingPong) {
		if v == 0 {
			return
		}
		pp.timeout = v
	}
}

func WithMessage(v string) Option {
	return func(pp *PingPong) {
		if v == "" {
			return
		}
		pp.message = v
	}
}
