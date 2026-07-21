package ops

import (
	"context"
	"errors"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/mxtypes"
)

// errTestWrite is returned by failing test response writers.
var errTestWrite = errors.New("write failed")

// quietLog returns an ExtendedLogger that suppresses everything below fatal,
// so tests that exercise warn/info/error log paths stay silent.
func quietLog() logger.ExtendedLogger {
	return logger.NewExtended(logger.WithLogLevel(logger.LogLevelFatal))
}

// fakeHealthChecker is a controllable mxtypes.HealthChecker.
type fakeHealthChecker struct {
	name     string
	interval time.Duration
	healthy  func(ctx context.Context) error
}

func (f *fakeHealthChecker) Name() string            { return f.name }
func (f *fakeHealthChecker) Interval() time.Duration { return f.interval }
func (f *fakeHealthChecker) Healthy(ctx context.Context) error {
	if f.healthy == nil {
		return nil
	}
	return f.healthy(ctx)
}

var _ mxtypes.HealthChecker = (*fakeHealthChecker)(nil)

// fakeStateProvider is a static mxtypes.StateProvider.
type fakeStateProvider struct {
	name  string
	state mxtypes.ServiceState
}

func (f fakeStateProvider) Name() string                { return f.name }
func (f fakeStateProvider) State() mxtypes.ServiceState { return f.state }

var _ mxtypes.StateProvider = (*fakeStateProvider)(nil)
