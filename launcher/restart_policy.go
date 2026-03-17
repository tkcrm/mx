package launcher

import "time"

// RestartMode defines when a service should be restarted after exit.
type RestartMode int

const (
	// RestartNever disables automatic restarts (default).
	RestartNever RestartMode = iota
	// RestartOnFailure restarts the service only when it exits with an error.
	RestartOnFailure
	// RestartAlways restarts the service on any exit (error or clean).
	RestartAlways
)

// RestartPolicy configures automatic restart behaviour for a service.
type RestartPolicy struct {
	// Mode controls when restarts happen.
	Mode RestartMode

	// MaxRetries is the maximum number of restart attempts. 0 means unlimited.
	MaxRetries int

	// Delay is the initial wait before the first restart attempt.
	Delay time.Duration

	// MaxDelay caps the exponential backoff. Zero defaults to Delay (no backoff growth).
	MaxDelay time.Duration
}

// nextDelay returns the backoff delay for attempt n (0-indexed) and whether
// another retry is allowed.
func (p RestartPolicy) nextDelay(attempt int) (time.Duration, bool) {
	if p.MaxRetries > 0 && attempt >= p.MaxRetries {
		return 0, false
	}

	delay := p.Delay
	if delay <= 0 {
		delay = time.Second
	}

	// exponential backoff: delay * 2^attempt
	for i := 0; i < attempt; i++ {
		delay *= 2
		if p.MaxDelay > 0 && delay > p.MaxDelay {
			delay = p.MaxDelay
			break
		}
	}

	return delay, true
}
