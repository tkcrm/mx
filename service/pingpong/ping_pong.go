package pingpong

import (
	"context"
	"sync"
	"time"
)

var defaultPingPongTimeout = time.Minute * 5

type PingPong struct {
	log     logger
	timeout time.Duration

	once sync.Once
	done chan struct{}
}

func New(logger logger, timeout time.Duration) *PingPong {
	if timeout == 0 {
		timeout = defaultPingPongTimeout
	}

	return &PingPong{log: logger, timeout: timeout, done: make(chan struct{})}
}

// Name of the service.
func (p *PingPong) Name() string { return "ping-pong" }

// Start ping-pong service.
func (p *PingPong) Start(ctx context.Context) error {
	timer := time.NewTimer(p.timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-p.done:
			return nil
		case <-timer.C:
			p.log.Info("ping-pong")
			timer.Reset(p.timeout)
		}
	}
}

// Stop ping-pong service.
func (p *PingPong) Stop(_ context.Context) error {
	p.once.Do(func() {
		close(p.done)
	})
	return nil
}
