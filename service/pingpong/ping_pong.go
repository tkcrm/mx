package pingpong

import (
	"context"
	"sync"
	"time"

	"github.com/tkcrm/mx/service"
)

var (
	defaultTimeout = time.Minute * 5
	defaultMessage = "ping-pong"
)

type PingPong struct {
	log logger

	// default: 5 minute
	timeout time.Duration
	// default: ping-pong
	message string

	once sync.Once
	done chan struct{}
}

// New return ping pong instance with default timeout 5 min
// and message ping-pong
func New(logger logger, opts ...Option) *PingPong {
	pp := &PingPong{log: logger, done: make(chan struct{})}
	for _, o := range opts {
		o(pp)
	}

	if pp.timeout == 0 {
		pp.timeout = defaultTimeout
	}

	if pp.message == "" {
		pp.message = defaultMessage
	}

	return pp
}

// Name of the service
func (p *PingPong) Name() string { return "ping-pong" }

// Start ping-pong service
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
			p.log.Info(p.message)
			timer.Reset(p.timeout)
		}
	}
}

// Stop ping-pong service
func (p *PingPong) Stop(_ context.Context) error {
	p.once.Do(func() {
		close(p.done)
	})
	return nil
}

var _ service.IService = (*PingPong)(nil)
