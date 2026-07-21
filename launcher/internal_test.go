package launcher

import (
	"context"
	"testing"
	"time"

	"github.com/tkcrm/mx/logger"
)

func quietLogger() logger.Logger {
	return logger.New(logger.WithLogLevel(logger.LogLevelFatal))
}

func TestRestartPolicy_NextDelay(t *testing.T) {
	tests := []struct {
		name      string
		policy    RestartPolicy
		attempt   int
		wantDelay time.Duration
		wantAllow bool
	}{
		{
			name:      "first attempt uses initial delay",
			policy:    RestartPolicy{Delay: time.Second},
			attempt:   0,
			wantDelay: time.Second,
			wantAllow: true,
		},
		{
			name:      "zero delay defaults to one second",
			policy:    RestartPolicy{},
			attempt:   0,
			wantDelay: time.Second,
			wantAllow: true,
		},
		{
			name:      "exponential growth without cap",
			policy:    RestartPolicy{Delay: time.Second},
			attempt:   2,
			wantDelay: 4 * time.Second,
			wantAllow: true,
		},
		{
			name:      "max delay caps growth",
			policy:    RestartPolicy{Delay: time.Second, MaxDelay: 3 * time.Second},
			attempt:   5,
			wantDelay: 3 * time.Second,
			wantAllow: true,
		},
		{
			name:      "max retries reached forbids further",
			policy:    RestartPolicy{Delay: time.Second, MaxRetries: 2},
			attempt:   2,
			wantDelay: 0,
			wantAllow: false,
		},
		{
			name:      "below max retries still allowed",
			policy:    RestartPolicy{Delay: time.Second, MaxRetries: 2},
			attempt:   1,
			wantDelay: 2 * time.Second,
			wantAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay, allowed := tt.policy.nextDelay(tt.attempt)
			if allowed != tt.wantAllow {
				t.Fatalf("allowed = %v; want %v", allowed, tt.wantAllow)
			}
			if delay != tt.wantDelay {
				t.Fatalf("delay = %v; want %v", delay, tt.wantDelay)
			}
		})
	}
}

// internalHC implements the full service + health checker surface.
type internalHC struct{ name string }

func (s *internalHC) Name() string                    { return s.name }
func (s *internalHC) Start(ctx context.Context) error { <-ctx.Done(); return nil }
func (s *internalHC) Stop(_ context.Context) error    { return nil }
func (s *internalHC) Interval() time.Duration         { return time.Second }
func (s *internalHC) Healthy(_ context.Context) error { return nil }

func TestServicesRunner_HCAndStateProviders(t *testing.T) {
	runner := newServicesRunner(context.Background(), quietLogger())

	withHC := NewService(WithService(&internalHC{name: "with-hc"}))
	plain := NewService(
		WithServiceName("plain"),
		WithStart(func(ctx context.Context) error { <-ctx.Done(); return nil }),
		WithStop(func(_ context.Context) error { return nil }),
	)
	runner.Register(withHC, plain)

	hcs := runner.hcServices()
	if len(hcs) != 1 {
		t.Fatalf("hcServices len = %d; want 1", len(hcs))
	}
	if hcs[0].Name() != "with-hc" {
		t.Errorf("hc name = %q; want with-hc", hcs[0].Name())
	}

	sps := runner.stateProviders()
	if len(sps) != 2 {
		t.Fatalf("stateProviders len = %d; want 2", len(sps))
	}
	// Every registered service must be exposed as a queryable state provider.
	names := map[string]bool{}
	for _, sp := range sps {
		names[sp.Name()] = true
		_ = sp.State() // must not panic
	}
	if !names["with-hc"] || !names["plain"] {
		t.Errorf("stateProviders missing services; got %v", names)
	}
}

func TestServicesRunner_RegisterValidationError_Skipped(t *testing.T) {
	runner := newServicesRunner(context.Background(), quietLogger())

	// Missing StopFn → Validate fails → service skipped.
	bad := NewService(
		WithServiceName("bad"),
		WithStart(func(ctx context.Context) error { <-ctx.Done(); return nil }),
	)
	runner.Register(bad)

	if len(runner.Services()) != 0 {
		t.Fatalf("invalid service should be skipped; got %d", len(runner.Services()))
	}
}
