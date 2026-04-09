package launcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// newTestLauncher creates a launcher suitable for synctest (signals disabled).
func newTestLauncher(opts ...Option) ILauncher {
	defaults := []Option{WithSignal(false)}
	return New(append(defaults, opts...)...)
}

// blockingStart is a StartFn that blocks until ctx is cancelled.
func blockingStart(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// noopStop is a StopFn that does nothing.
func noopStop(_ context.Context) error { return nil }

// --- Service lifecycle tests ---

func TestService_StartStop_Basic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := NewService(
			WithServiceName("basic"),
			WithStart(blockingStart),
			WithStop(noopStop),
		)

		if svc.State() != ServiceStateIdle {
			t.Fatalf("initial state = %v; want idle", svc.State())
		}

		ctx, cancel := context.WithCancel(context.Background())
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		synctest.Wait()

		if svc.State() != ServiceStateRunning {
			t.Fatalf("after start, state = %v; want running", svc.State())
		}

		select {
		case <-svc.Ready():
		default:
			t.Fatal("Ready() channel not closed after start")
		}

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v", err)
		}

		if err := svc.Stop(); err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}

		if svc.State() != ServiceStateStopped {
			t.Fatalf("after stop, state = %v; want stopped", svc.State())
		}
	})
}

func TestService_NilStartFn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := NewService(WithServiceName("nil-start"))
		if err := svc.Start(); err != nil {
			t.Fatalf("Start with nil StartFn should return nil, got: %v", err)
		}
		select {
		case <-svc.Ready():
		default:
			t.Fatal("Ready() not closed after nil StartFn")
		}
	})
}

func TestService_Disabled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := false
		svc := NewService(
			WithServiceName("disabled"),
			WithStart(func(ctx context.Context) error {
				started = true
				<-ctx.Done()
				return nil
			}),
			WithStop(noopStop),
			WithEnabled(false),
		)
		svc.Options().Context = context.Background()

		if err := svc.Start(); err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
		if started {
			t.Fatal("disabled service should not have started")
		}
	})
}

func TestService_DoubleStart_Ignored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var startCount atomic.Int32
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := NewService(
			WithServiceName("double"),
			WithStart(func(ctx context.Context) error {
				startCount.Add(1)
				<-ctx.Done()
				return nil
			}),
			WithStop(noopStop),
		)
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		if err := svc.Start(); err != nil {
			t.Fatalf("second Start returned error: %v", err)
		}

		if startCount.Load() != 1 {
			t.Fatalf("StartFn called %d times; want 1", startCount.Load())
		}
	})
}

func TestService_StopIdempotent(t *testing.T) {
	svc := NewService(
		WithServiceName("idempotent-stop"),
		WithStart(blockingStart),
		WithStop(noopStop),
	)
	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop on idle service returned error: %v", err)
	}
}

// --- Hooks tests ---

func TestService_BeforeStartHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("before start failed")
		svc := NewService(
			WithServiceName("hook-err"),
			WithStart(blockingStart),
			WithStop(noopStop),
			WithServiceBeforeStart(func() error { return hookErr }),
		)
		svc.Options().Context = context.Background()

		err := svc.Start()
		if !errors.Is(err, hookErr) {
			t.Fatalf("Start error = %v; want %v", err, hookErr)
		}
		if svc.State() != ServiceStateFailed {
			t.Fatalf("state = %v; want failed", svc.State())
		}

	})
}

func TestService_AfterStartHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("after start failed")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := NewService(
			WithServiceName("after-hook-err"),
			WithStart(blockingStart),
			WithStop(noopStop),
			WithServiceAfterStart(func() error { return hookErr }),
		)
		svc.Options().Context = ctx

		err := svc.Start()
		if !errors.Is(err, hookErr) {
			t.Fatalf("Start error = %v; want %v", err, hookErr)
		}
		if svc.State() != ServiceStateFailed {
			t.Fatalf("state = %v; want failed", svc.State())
		}
	})
}

func TestService_BeforeStopAfterStop_Hooks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var order []string

		ctx, cancel := context.WithCancel(context.Background())

		svc := NewService(
			WithServiceName("stop-hooks"),
			WithStart(blockingStart),
			WithStop(noopStop),
			WithServiceBeforeStop(func() error {
				order = append(order, "before")
				return nil
			}),
			WithServiceAfterStop(func() error {
				order = append(order, "after")
				return nil
			}),
		)
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := svc.Stop(); err != nil {
			t.Fatalf("Stop error: %v", err)
		}

		if len(order) != 2 || order[0] != "before" || order[1] != "after" {
			t.Fatalf("hook order = %v; want [before after]", order)
		}
	})
}

func TestService_AfterStartFinished_Hook(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		called := false
		svc := NewService(
			WithServiceName("after-finished"),
			WithStart(func(_ context.Context) error { return nil }),
			WithStop(noopStop),
			WithServiceAfterStartFinished(func() error {
				called = true
				return nil
			}),
		)
		svc.Options().Context = context.Background()

		if err := svc.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		if !called {
			t.Fatal("AfterStartFinished hook was not called")
		}
	})
}

// --- Startup timeout ---

func TestService_StartupTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := NewService(
			WithServiceName("slow-start"),
			WithStart(func(ctx context.Context) error {
				// Never finishes on its own — only stops via context cancel
				<-ctx.Done()
				return nil
			}),
			WithStop(noopStop),
			WithStartupTimeout(5*time.Second),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		synctest.Wait()

		err := <-errCh
		if err == nil {
			t.Fatal("expected startup timeout error")
		}
		if svc.State() != ServiceStateFailed {
			t.Fatalf("state = %v; want failed", svc.State())
		}
	})
}

// --- Shutdown timeout ---

func TestService_ShutdownTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := NewService(
			WithServiceName("slow-stop"),
			WithStart(blockingStart),
			WithStop(func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			}),
			WithShutdownTimeout(3*time.Second),
		)

		ctx, cancel := context.WithCancel(context.Background())
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := svc.Stop(); err != nil {
			t.Fatalf("Stop error: %v", err)
		}
		if svc.State() != ServiceStateStopped {
			t.Fatalf("state = %v; want stopped", svc.State())
		}
	})
}

// --- Restart policy ---

func TestService_RestartOnFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int32

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		svc := NewService(
			WithServiceName("restart-fail"),
			WithStart(func(ctx context.Context) error {
				n := attempts.Add(1)
				if n <= 2 {
					return errors.New("transient error")
				}
				<-ctx.Done()
				return nil
			}),
			WithStop(noopStop),
			WithRestartPolicy(RestartPolicy{
				Mode:     RestartOnFailure,
				Delay:    time.Second,
				MaxDelay: 4 * time.Second,
			}),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		// Advance fake time past all backoff delays (1s + 2s = 3s)
		time.Sleep(10 * time.Second)
		synctest.Wait()

		if svc.State() != ServiceStateRunning {
			t.Fatalf("state = %v; want running", svc.State())
		}
		if attempts.Load() != 3 {
			t.Fatalf("attempts = %d; want 3", attempts.Load())
		}

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	})
}

func TestService_RestartOnFailure_CleanExit_NoRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int32

		svc := NewService(
			WithServiceName("restart-clean"),
			WithStart(func(_ context.Context) error {
				attempts.Add(1)
				return nil
			}),
			WithStop(noopStop),
			WithRestartPolicy(RestartPolicy{
				Mode:  RestartOnFailure,
				Delay: time.Second,
			}),
		)
		svc.Options().Context = context.Background()

		if err := svc.Start(); err != nil {
			t.Fatalf("Start error: %v", err)
		}
		if attempts.Load() != 1 {
			t.Fatalf("attempts = %d; want 1 (no restart on clean exit)", attempts.Load())
		}
	})
}

func TestService_RestartAlways(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int32

		ctx, cancel := context.WithCancel(context.Background())

		svc := NewService(
			WithServiceName("restart-always"),
			WithStart(func(ctx context.Context) error {
				n := attempts.Add(1)
				if n < 3 {
					return nil
				}
				<-ctx.Done()
				return nil
			}),
			WithStop(noopStop),
			WithRestartPolicy(RestartPolicy{
				Mode:  RestartAlways,
				Delay: time.Second,
			}),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		// Advance fake time past backoff delays (1s + 2s = 3s)
		time.Sleep(10 * time.Second)
		synctest.Wait()

		if attempts.Load() != 3 {
			t.Fatalf("attempts = %d; want 3", attempts.Load())
		}

		cancel()
		synctest.Wait()
		<-errCh
	})
}

func TestService_RestartMaxRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int32
		failErr := errors.New("always fail")

		svc := NewService(
			WithServiceName("max-retries"),
			WithStart(func(_ context.Context) error {
				attempts.Add(1)
				return failErr
			}),
			WithStop(noopStop),
			WithRestartPolicy(RestartPolicy{
				Mode:       RestartOnFailure,
				MaxRetries: 3,
				Delay:      time.Second,
			}),
		)
		svc.Options().Context = context.Background()

		err := svc.Start()
		if err == nil {
			t.Fatal("expected error after max retries")
		}
		// 1 initial + 3 retries = 4 total attempts
		if attempts.Load() != 4 {
			t.Fatalf("attempts = %d; want 4", attempts.Load())
		}
		if svc.State() != ServiceStateFailed {
			t.Fatalf("state = %v; want failed", svc.State())
		}
	})
}

func TestService_RestartBackoff_ExponentialDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var timestamps []time.Time

		ctx, cancel := context.WithCancel(context.Background())

		svc := NewService(
			WithServiceName("backoff"),
			WithStart(func(ctx context.Context) error {
				timestamps = append(timestamps, time.Now())
				if len(timestamps) < 4 {
					return errors.New("fail")
				}
				<-ctx.Done()
				return nil
			}),
			WithStop(noopStop),
			WithRestartPolicy(RestartPolicy{
				Mode:     RestartOnFailure,
				Delay:    time.Second,
				MaxDelay: 10 * time.Second,
			}),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		// Advance fake time past all backoff delays (1s + 2s + 4s = 7s)
		time.Sleep(10 * time.Second)
		synctest.Wait()

		cancel()
		synctest.Wait()
		<-errCh

		if len(timestamps) != 4 {
			t.Fatalf("got %d attempts; want 4", len(timestamps))
		}

		// Delays: attempt 0→1: 1s, attempt 1→2: 2s, attempt 2→3: 4s
		d1 := timestamps[1].Sub(timestamps[0])
		d2 := timestamps[2].Sub(timestamps[1])
		d3 := timestamps[3].Sub(timestamps[2])

		if d1 != time.Second {
			t.Fatalf("delay 1 = %v; want 1s", d1)
		}
		if d2 != 2*time.Second {
			t.Fatalf("delay 2 = %v; want 2s", d2)
		}
		if d3 != 4*time.Second {
			t.Fatalf("delay 3 = %v; want 4s", d3)
		}
	})
}

// --- Launcher tests ---

func TestLauncher_RunStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ln := newTestLauncher()

		svc := NewService(
			WithServiceName("svc"),
			WithStart(blockingStart),
			WithStop(noopStop),
		)
		ln.ServicesRunner().Register(svc)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()

		synctest.Wait()

		if svc.State() != ServiceStateRunning {
			t.Fatalf("state = %v; want running", svc.State())
		}

		ln.Stop()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
}

func TestLauncher_NoServices(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ln := newTestLauncher()

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()

		synctest.Wait()

		ln.Stop()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Run with no services returned error: %v", err)
		}
	})
}

func TestLauncher_ServiceError_Propagates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svcErr := errors.New("service crashed")
		ln := newTestLauncher()

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("crasher"),
				WithStart(func(_ context.Context) error { return svcErr }),
				WithStop(noopStop),
			),
		)

		err := ln.Run()
		if err == nil {
			t.Fatal("expected error from Run")
		}
		if !errors.Is(err, svcErr) {
			t.Fatalf("Run error = %v; want wrapping %v", err, svcErr)
		}
	})
}

func TestLauncher_BeforeStartHook_Error(t *testing.T) {
	hookErr := errors.New("before start hook failed")
	ln := newTestLauncher(
		WithBeforeStart(func() error { return hookErr }),
	)

	ln.ServicesRunner().Register(
		NewService(
			WithServiceName("svc"),
			WithStart(blockingStart),
			WithStop(noopStop),
		),
	)

	err := ln.Run()
	if !errors.Is(err, hookErr) {
		t.Fatalf("Run error = %v; want %v", err, hookErr)
	}
}

func TestLauncher_AfterStartHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("after start hook failed")
		ln := newTestLauncher(
			WithAfterStart(func() error { return hookErr }),
		)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		err := ln.Run()
		if !errors.Is(err, hookErr) {
			t.Fatalf("Run error = %v; want %v", err, hookErr)
		}
	})
}

func TestLauncher_BeforeStopAfterStop_Hooks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var order []string
		var mu sync.Mutex

		appendOrder := func(s string) {
			mu.Lock()
			order = append(order, s)
			mu.Unlock()
		}

		ln := newTestLauncher(
			WithBeforeStop(func() error { appendOrder("before-stop"); return nil }),
			WithAfterStop(func() error { appendOrder("after-stop"); return nil }),
		)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Run error: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()
		if len(order) != 2 || order[0] != "before-stop" || order[1] != "after-stop" {
			t.Fatalf("hook order = %v; want [before-stop after-stop]", order)
		}
	})
}

func TestLauncher_AddHooksAfterCreation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var called []string

		ln := newTestLauncher()
		ln.AddBeforeStartHooks(func() error { called = append(called, "before"); return nil })
		ln.AddAfterStartHooks(func() error { called = append(called, "after-start"); return nil })
		ln.AddBeforeStopHooks(func() error { called = append(called, "before-stop"); return nil })
		ln.AddAfterStopHooks(func() error { called = append(called, "after-stop"); return nil })

		// Nil hooks should be ignored
		ln.AddBeforeStartHooks(nil)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()
		<-errCh

		want := []string{"before", "after-start", "before-stop", "after-stop"}
		if len(called) != len(want) {
			t.Fatalf("hooks = %v; want %v", called, want)
		}
		for i := range want {
			if called[i] != want[i] {
				t.Fatalf("hooks[%d] = %q; want %q", i, called[i], want[i])
			}
		}
	})
}

func TestLauncher_ContextCancel_StopsLauncher(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ln := newTestLauncher(WithContext(ctx))

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Run error: %v", err)
		}
	})
}

// --- Stop sequence tests ---

func TestLauncher_StopSequence_Fifo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var order []string
		var mu sync.Mutex

		ln := newTestLauncher(
			WithRunnerServicesSequence(RunnerServicesSequenceFifo),
		)

		for _, name := range []string{"a", "b", "c"} {
			n := name
			ln.ServicesRunner().Register(
				NewService(
					WithServiceName(n),
					WithStart(blockingStart),
					WithStop(func(_ context.Context) error {
						mu.Lock()
						order = append(order, n)
						mu.Unlock()
						return nil
					}),
				),
			)
		}

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()
		<-errCh

		mu.Lock()
		defer mu.Unlock()
		want := []string{"a", "b", "c"}
		if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
			t.Fatalf("stop order = %v; want %v", order, want)
		}
	})
}

func TestLauncher_StopSequence_Lifo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var order []string
		var mu sync.Mutex

		ln := newTestLauncher(
			WithRunnerServicesSequence(RunnerServicesSequenceLifo),
		)

		for _, name := range []string{"a", "b", "c"} {
			n := name
			ln.ServicesRunner().Register(
				NewService(
					WithServiceName(n),
					WithStart(blockingStart),
					WithStop(func(_ context.Context) error {
						mu.Lock()
						order = append(order, n)
						mu.Unlock()
						return nil
					}),
				),
			)
		}

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()
		<-errCh

		mu.Lock()
		defer mu.Unlock()
		want := []string{"c", "b", "a"}
		if len(order) != 3 || order[0] != "c" || order[1] != "b" || order[2] != "a" {
			t.Fatalf("stop order = %v; want %v", order, want)
		}
	})
}

func TestLauncher_StopSequence_None_Parallel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var stopCount atomic.Int32

		ln := newTestLauncher(
			WithRunnerServicesSequence(RunnerServicesSequenceNone),
		)

		for _, name := range []string{"a", "b", "c"} {
			ln.ServicesRunner().Register(
				NewService(
					WithServiceName(name),
					WithStart(blockingStart),
					WithStop(func(_ context.Context) error {
						stopCount.Add(1)
						return nil
					}),
				),
			)
		}

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()
		<-errCh

		if stopCount.Load() != 3 {
			t.Fatalf("stopped %d services; want 3", stopCount.Load())
		}
	})
}

// --- Startup priority tests ---

func TestLauncher_StartupPriority_GroupOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var order []string
		var mu sync.Mutex

		appendOrder := func(s string) {
			mu.Lock()
			order = append(order, s)
			mu.Unlock()
		}

		ln := newTestLauncher()

		// Priority 2
		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("broker"),
				WithStartupPriority(2),
				WithStart(func(ctx context.Context) error {
					appendOrder("broker-started")
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			),
		)

		// Priority 1
		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("db"),
				WithStartupPriority(1),
				WithStart(func(ctx context.Context) error {
					appendOrder("db-started")
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			),
		)

		// Priority 0 (default)
		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("http"),
				WithStart(func(ctx context.Context) error {
					appendOrder("http-started")
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()

		// Advance fake time past the inter-group yield sleeps
		time.Sleep(time.Millisecond)
		synctest.Wait()

		ln.Stop()
		synctest.Wait()
		<-errCh

		mu.Lock()
		defer mu.Unlock()

		if len(order) != 3 {
			t.Fatalf("order = %v; want 3 entries", order)
		}
		if order[0] != "db-started" {
			t.Fatalf("order[0] = %q; want db-started", order[0])
		}
		if order[1] != "broker-started" {
			t.Fatalf("order[1] = %q; want broker-started", order[1])
		}
		if order[2] != "http-started" {
			t.Fatalf("order[2] = %q; want http-started", order[2])
		}
	})
}

func TestLauncher_StartupPriority_SameGroup_Concurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var readyCount atomic.Int32
		bothReady := make(chan struct{})

		ln := newTestLauncher()

		makeSvc := func(name string) *Service {
			return NewService(
				WithServiceName(name),
				WithStartupPriority(1),
				WithStart(func(ctx context.Context) error {
					if readyCount.Add(1) == 2 {
						close(bothReady)
					}
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			)
		}

		ln.ServicesRunner().Register(makeSvc("pg"), makeSvc("redis"))

		var p0Started atomic.Bool
		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("app"),
				WithStart(func(ctx context.Context) error {
					p0Started.Store(true)
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()

		// Advance fake time past inter-group yield sleeps
		time.Sleep(time.Millisecond)
		synctest.Wait()

		select {
		case <-bothReady:
		default:
			t.Fatal("both priority-1 services should have started concurrently")
		}

		if !p0Started.Load() {
			t.Fatal("priority-0 service should have started after group 1")
		}

		ln.Stop()
		synctest.Wait()
		<-errCh
	})
}

func TestLauncher_StartupPriority_GroupFailure_AbortsNext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svcErr := errors.New("db failed")
		var p2Started atomic.Bool

		ln := newTestLauncher()

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("db"),
				WithStartupPriority(1),
				WithStart(func(_ context.Context) error { return svcErr }),
				WithStop(noopStop),
			),
		)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("broker"),
				WithStartupPriority(2),
				WithStart(func(ctx context.Context) error {
					p2Started.Store(true)
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			),
		)

		err := ln.Run()
		if err == nil {
			t.Fatal("expected error from Run")
		}
		if !errors.Is(err, svcErr) {
			t.Fatalf("Run error = %v; want wrapping %v", err, svcErr)
		}
		if p2Started.Load() {
			t.Fatal("priority-2 service should not have started after priority-1 failure")
		}
	})
}

func TestLauncher_StartupPriority_AllDefault_Concurrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var startCount atomic.Int32
		allStarted := make(chan struct{})
		const n = 5

		ln := newTestLauncher()

		for i := range n {
			ln.ServicesRunner().Register(
				NewService(
					WithServiceName(fmt.Sprintf("svc-%d", i)),
					WithStart(func(ctx context.Context) error {
						if startCount.Add(1) == n {
							close(allStarted)
						}
						<-ctx.Done()
						return nil
					}),
					WithStop(noopStop),
				),
			)
		}

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		select {
		case <-allStarted:
		default:
			t.Fatalf("expected all %d services to start concurrently, got %d", n, startCount.Load())
		}

		ln.Stop()
		synctest.Wait()
		<-errCh
	})
}

func TestLauncher_StartupPriority_MultipleGroups(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var seq atomic.Int64
		var g1Seq, g2Seq, g3Seq int64

		ln := newTestLauncher()

		makeGroupSvc := func(name string, priority int, seqDst *int64) *Service {
			return NewService(
				WithServiceName(name),
				WithStartupPriority(priority),
				WithStart(func(ctx context.Context) error {
					*seqDst = seq.Add(1)
					<-ctx.Done()
					return nil
				}),
				WithStop(noopStop),
			)
		}

		ln.ServicesRunner().Register(
			makeGroupSvc("g3-svc", 3, &g3Seq),
			makeGroupSvc("g1-svc", 1, &g1Seq),
			makeGroupSvc("g2-svc", 2, &g2Seq),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()

		time.Sleep(time.Millisecond)
		synctest.Wait()

		if g1Seq >= g2Seq || g2Seq >= g3Seq {
			t.Fatalf("groups out of order: g1=%d, g2=%d, g3=%d", g1Seq, g2Seq, g3Seq)
		}

		ln.Stop()
		synctest.Wait()
		<-errCh
	})
}

// --- Services runner tests ---

func TestServicesRunner_Get(t *testing.T) {
	ln := newTestLauncher()

	svc := NewService(
		WithServiceName("findme"),
		WithStart(blockingStart),
		WithStop(noopStop),
	)
	ln.ServicesRunner().Register(svc)

	found, ok := ln.ServicesRunner().Get("findme")
	if !ok {
		t.Fatal("service not found")
	}
	if found.Name() != "findme" {
		t.Fatalf("found service name = %q; want findme", found.Name())
	}

	_, ok = ln.ServicesRunner().Get("nonexistent")
	if ok {
		t.Fatal("nonexistent service should not be found")
	}
}

func TestServicesRunner_RegisterNil(t *testing.T) {
	ln := newTestLauncher()
	ln.ServicesRunner().Register(nil)

	if len(ln.ServicesRunner().Services()) != 0 {
		t.Fatal("nil service should not be registered")
	}
}

func TestServicesRunner_DisabledService_Skipped(t *testing.T) {
	ln := newTestLauncher()

	ln.ServicesRunner().Register(
		NewService(
			WithServiceName("disabled"),
			WithStart(blockingStart),
			WithStop(noopStop),
			WithEnabled(false),
		),
	)

	if len(ln.ServicesRunner().Services()) != 0 {
		t.Fatal("disabled service should not be registered")
	}
}

// --- Multiple services with errors ---

func TestLauncher_MultipleServices_OneFailsAfterStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svcErr := errors.New("service-b crashed")

		ln := newTestLauncher()

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("a"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
			NewService(
				WithServiceName("b"),
				WithStart(func(ctx context.Context) error {
					time.Sleep(time.Second)
					return svcErr
				}),
				WithStop(noopStop),
			),
		)

		err := ln.Run()
		if err == nil {
			t.Fatal("expected error from Run")
		}
		if !errors.Is(err, svcErr) {
			t.Fatalf("Run error = %v; want wrapping %v", err, svcErr)
		}
	})
}

// --- Stop error from hooks ---

func TestLauncher_BeforeStopHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("before stop failed")
		ln := newTestLauncher(
			WithBeforeStop(func() error { return hookErr }),
		)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()

		err := <-errCh
		if !errors.Is(err, hookErr) {
			t.Fatalf("Run error = %v; want wrapping %v", err, hookErr)
		}
	})
}

func TestLauncher_AfterStopHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("after stop failed")
		ln := newTestLauncher(
			WithAfterStop(func() error { return hookErr }),
		)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()

		err := <-errCh
		if !errors.Is(err, hookErr) {
			t.Fatalf("Run error = %v; want wrapping %v", err, hookErr)
		}
	})
}

// --- Service stop error ---

func TestService_StopError_Propagates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stopErr := errors.New("stop failed")
		ctx, cancel := context.WithCancel(context.Background())

		svc := NewService(
			WithServiceName("stop-err"),
			WithStart(blockingStart),
			WithStop(func(_ context.Context) error { return stopErr }),
		)
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		err := svc.Stop()
		if !errors.Is(err, stopErr) {
			t.Fatalf("Stop error = %v; want %v", err, stopErr)
		}
		if svc.State() != ServiceStateFailed {
			t.Fatalf("state = %v; want failed", svc.State())
		}
	})
}

// --- Ready channel edge cases ---

func TestService_ReadyCh_ClosedOnStartError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := NewService(
			WithServiceName("fail-start"),
			WithStart(func(_ context.Context) error { return errors.New("boom") }),
			WithStop(noopStop),
		)
		svc.Options().Context = context.Background()

		_ = svc.Start()

		select {
		case <-svc.Ready():
		default:
			t.Fatal("Ready() should be closed even after StartFn error")
		}
	})
}

// --- WithService wrapper ---

type testIService struct {
	name    string
	started atomic.Bool
	stopped atomic.Bool
}

func (s *testIService) Name() string                    { return s.name }
func (s *testIService) Start(ctx context.Context) error { s.started.Store(true); <-ctx.Done(); return nil }
func (s *testIService) Stop(_ context.Context) error    { s.stopped.Store(true); return nil }
func (s *testIService) Enabled() bool                   { return true }
func (s *testIService) Interval() time.Duration         { return time.Second }
func (s *testIService) Healthy(_ context.Context) error { return nil }

func TestService_WithServiceWrapper(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		impl := &testIService{name: "wrapped"}

		svc := NewService(WithService(impl))

		if svc.Name() != "wrapped" {
			t.Fatalf("name = %q; want wrapped", svc.Name())
		}

		ctx, cancel := context.WithCancel(context.Background())
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		if !impl.started.Load() {
			t.Fatal("Start was not called on underlying service")
		}

		cancel()
		synctest.Wait()

		_ = svc.Stop()
		if !impl.stopped.Load() {
			t.Fatal("Stop was not called on underlying service")
		}
	})
}

// --- Launcher context ---

func TestLauncher_Context(t *testing.T) {
	ln := newTestLauncher()
	ctx := ln.Context()
	if ctx == nil {
		t.Fatal("Context() returned nil")
	}
}

// --- AppStartStopLog ---

func TestLauncher_AppStartStopLog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ln := newTestLauncher(
			WithName("test-app"),
			WithAppStartStopLog(true),
		)

		ln.ServicesRunner().Register(
			NewService(
				WithServiceName("svc"),
				WithStart(blockingStart),
				WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Run error: %v", err)
		}
	})
}
