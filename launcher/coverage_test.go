package launcher_test

import (
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/launcher/ops"
	"github.com/tkcrm/mx/logger"
)

func quietExtended() logger.ExtendedLogger {
	return logger.NewExtended(logger.WithLogLevel(logger.LogLevelFatal))
}

// --- Small exported surface ---

func TestShutdownSignal(t *testing.T) {
	sigs := launcher.ShutdownSiganl()
	found := map[string]bool{}
	for _, s := range sigs {
		found[s.String()] = true
	}
	for _, s := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT} {
		if !found[s.String()] {
			t.Errorf("ShutdownSiganl missing %s", s)
		}
	}
}

func TestService_String(t *testing.T) {
	svc := launcher.NewService(launcher.WithServiceName("x"))
	if svc.String() != "mx" {
		t.Errorf("String() = %q; want mx", svc.String())
	}
}

func TestWithServiceContextAndLogger(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{ k string }{"k"}, "v")
	l := quietExtended()

	svc := launcher.NewService(
		launcher.WithServiceName("x"),
		launcher.WithServiceContext(ctx),
		launcher.WithServiceLogger(l),
	)

	if svc.Options().Context != ctx {
		t.Error("WithServiceContext did not set context")
	}
	if svc.Options().Logger != l {
		t.Error("WithServiceLogger did not set logger")
	}
}

func TestServiceOptions_Validate(t *testing.T) {
	l := logger.New(logger.WithLogLevel(logger.LogLevelFatal))
	start := func(context.Context) error { return nil }
	stop := func(context.Context) error { return nil }
	ctx := context.Background()

	base := func() launcher.ServiceOptions {
		return launcher.ServiceOptions{
			Logger:  l,
			Name:    "svc",
			StartFn: start,
			StopFn:  stop,
			Context: ctx,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*launcher.ServiceOptions)
		wantSub string // empty → expect success
	}{
		{"valid", func(*launcher.ServiceOptions) {}, ""},
		{"nil logger", func(o *launcher.ServiceOptions) { o.Logger = nil }, "logger"},
		{"empty name", func(o *launcher.ServiceOptions) { o.Name = "" }, "name"},
		{"nil start", func(o *launcher.ServiceOptions) { o.StartFn = nil }, "Start"},
		{"nil stop", func(o *launcher.ServiceOptions) { o.StopFn = nil }, "Stop"},
		{"nil context", func(o *launcher.ServiceOptions) { o.Context = nil }, "context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := base()
			tt.mutate(&o)
			err := o.Validate()
			if tt.wantSub == "" {
				if err != nil {
					t.Fatalf("Validate() = %v; want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil; want error containing %q", tt.wantSub)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("Validate() = %q; want to contain %q", err, tt.wantSub)
			}
		})
	}
}

func TestLauncher_OptionsSmoke(t *testing.T) {
	// Executes the remaining option setters; no observable effect beyond no panic.
	ln := launcher.New(
		launcher.WithSignal(false),
		launcher.WithName("app"),
		launcher.WithVersion("v1.2.3"),
		launcher.WithLogger(quietExtended()),
		launcher.WithGlobalShutdownTimeout(5*time.Second),
	)
	if ln.Context() == nil {
		t.Fatal("Context() = nil")
	}
}

func TestLauncher_AddRemainingHooks_NilIgnored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ln := newTestLauncher()
		// Nil hooks must be ignored for every Add*Hooks variant.
		ln.AddBeforeStartHooks(nil)
		ln.AddBeforeStopHooks(nil)
		ln.AddAfterStartHooks(nil)
		ln.AddAfterStopHooks(nil)

		var called int
		ln.AddBeforeStopHooks(func() error { called++; return nil })
		ln.AddAfterStartHooks(func() error { called++; return nil })
		ln.AddAfterStopHooks(func() error { called++; return nil })

		ln.ServicesRunner().Register(
			launcher.NewService(
				launcher.WithServiceName("svc"),
				launcher.WithStart(blockingStart),
				launcher.WithStop(noopStop),
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

		// Three non-nil hooks fired exactly once each; the four nil hooks were ignored.
		if called != 3 {
			t.Fatalf("hooks called = %d; want 3", called)
		}
	})
}

// --- Global shutdown timeout branch ---

func TestLauncher_GlobalShutdownTimeout_NormalStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ln := newTestLauncher(
			launcher.WithGlobalShutdownTimeout(30 * time.Second),
		)
		ln.ServicesRunner().Register(
			launcher.NewService(
				launcher.WithServiceName("svc"),
				launcher.WithStart(blockingStart),
				launcher.WithStop(noopStop),
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

// --- runWithRestarts extra branches ---

func TestService_StartupTimeout_ContextCancelBeforeTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		svc := launcher.NewService(
			launcher.WithServiceName("startup-timeout-cancel"),
			launcher.WithStart(blockingStart),
			launcher.WithStop(noopStop),
			launcher.WithStartupTimeout(1*time.Hour), // far in the future
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()
		synctest.Wait()

		cancel() // cancel well before the startup timeout fires
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v; want nil (context cancel, not timeout)", err)
		}
	})
}

func TestService_RestartBackoff_ContextCancelDuringDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		svc := launcher.NewService(
			launcher.WithServiceName("cancel-during-delay"),
			launcher.WithStart(func(context.Context) error { return errors.New("always fails") }),
			launcher.WithStop(noopStop),
			launcher.WithRestartPolicy(launcher.RestartPolicy{
				Mode:  launcher.RestartOnFailure,
				Delay: 1 * time.Hour, // long backoff so cancel lands during the wait
			}),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		// Let the first attempt fail and enter the backoff wait.
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v; want nil (cancelled during backoff)", err)
		}
	})
}

func TestService_RestartAlways_CleanExit_MaxRetriesExhausted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := launcher.NewService(
			launcher.WithServiceName("clean-exhausted"),
			launcher.WithStart(func(context.Context) error { return nil }), // always clean exit
			launcher.WithStop(noopStop),
			launcher.WithRestartPolicy(launcher.RestartPolicy{
				Mode:       launcher.RestartAlways,
				MaxRetries: 2,
				Delay:      time.Second,
			}),
		)
		svc.Options().Context = context.Background()

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()

		// Advance past both backoff delays (1s + 2s).
		time.Sleep(10 * time.Second)
		synctest.Wait()

		// Clean exit with retries exhausted → returns nil, not an error.
		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v; want nil", err)
		}
	})
}

// --- Stop hook error ---

func TestService_StopBeforeStopHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("before stop hook failed")
		ctx, cancel := context.WithCancel(context.Background())

		svc := launcher.NewService(
			launcher.WithServiceName("stop-hook-err"),
			launcher.WithStart(blockingStart),
			launcher.WithStop(noopStop),
			launcher.WithServiceBeforeStop(func() error { return hookErr }),
		)
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := svc.Stop(); !errors.Is(err, hookErr) {
			t.Fatalf("Stop error = %v; want %v", err, hookErr)
		}
	})
}

func TestService_StartupTimeout_StartError(t *testing.T) {
	errBoom := errors.New("boom")
	svc := launcher.NewService(
		launcher.WithServiceName("startup-err"),
		launcher.WithStart(func(context.Context) error { return errBoom }),
		launcher.WithStop(noopStop),
		launcher.WithStartupTimeout(time.Hour),
	)
	svc.Options().Context = context.Background()

	if err := svc.Start(); !errors.Is(err, errBoom) {
		t.Fatalf("Start error = %v; want %v", err, errBoom)
	}
	if svc.State() != launcher.ServiceStateFailed {
		t.Fatalf("state = %v; want failed", svc.State())
	}
}

func TestService_StartupTimeout_CleanExit(t *testing.T) {
	svc := launcher.NewService(
		launcher.WithServiceName("startup-clean"),
		launcher.WithStart(func(context.Context) error { return nil }),
		launcher.WithStop(noopStop),
		launcher.WithStartupTimeout(time.Hour),
	)
	svc.Options().Context = context.Background()

	if err := svc.Start(); err != nil {
		t.Fatalf("Start error = %v; want nil", err)
	}
}

// startFn that ignores ctx and only returns when release is closed.
func makeUnresponsiveStart(release <-chan struct{}) func(context.Context) error {
	return func(context.Context) error {
		<-release
		return nil
	}
}

func TestService_StartupTimeout_ShutdownTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())

		svc := launcher.NewService(
			launcher.WithServiceName("startup-shutdown-timeout"),
			launcher.WithStart(makeUnresponsiveStart(release)),
			launcher.WithStop(noopStop),
			launcher.WithStartupTimeout(time.Hour),
			launcher.WithShutdownTimeout(3*time.Second),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()
		synctest.Wait()

		cancel() // StartFn ignores ctx → shutdown timeout path fires
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start error = %v; want nil (stopped by timeout)", err)
		}
		close(release) // drain the unresponsive goroutine
		synctest.Wait()
	})
}

func TestService_NoStartupTimeout_ShutdownTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())

		svc := launcher.NewService(
			launcher.WithServiceName("noto-shutdown-timeout"),
			launcher.WithStart(makeUnresponsiveStart(release)),
			launcher.WithStop(noopStop),
			launcher.WithShutdownTimeout(3*time.Second),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start error = %v; want nil (stopped by timeout)", err)
		}
		close(release)
		synctest.Wait()
	})
}

func TestService_Stop_NilStopFn(t *testing.T) {
	svc := launcher.NewService(launcher.WithServiceName("no-stop"))
	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop with nil StopFn = %v; want nil", err)
	}
}

func TestService_Stop_AfterStopHook_Error(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hookErr := errors.New("after stop failed")
		ctx, cancel := context.WithCancel(context.Background())

		svc := launcher.NewService(
			launcher.WithServiceName("afterstop-err"),
			launcher.WithStart(blockingStart),
			launcher.WithStop(noopStop),
			launcher.WithServiceAfterStop(func() error { return hookErr }),
		)
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()
		cancel()
		synctest.Wait()

		if err := svc.Stop(); !errors.Is(err, hookErr) {
			t.Fatalf("Stop error = %v; want %v", err, hookErr)
		}
	})
}

// --- Stop-sequence error logging (None / Fifo / Lifo) ---

func runLauncherWithFailingStop(t *testing.T, seq launcher.RunnerServicesSequence) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		ln := newTestLauncher(launcher.WithRunnerServicesSequence(seq))
		ln.ServicesRunner().Register(
			launcher.NewService(
				launcher.WithServiceName("bad-stop"),
				launcher.WithStart(blockingStart),
				launcher.WithStop(func(context.Context) error { return errors.New("stop failed") }),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		ln.Stop()
		synctest.Wait()

		// Stop errors are logged, not returned by Run.
		if err := <-errCh; err != nil {
			t.Fatalf("Run error = %v; want nil (stop errors only logged)", err)
		}
	})
}

func TestLauncher_StopSequence_None_StopError(t *testing.T) {
	runLauncherWithFailingStop(t, launcher.RunnerServicesSequenceNone)
}

func TestLauncher_StopSequence_Fifo_StopError(t *testing.T) {
	runLauncherWithFailingStop(t, launcher.RunnerServicesSequenceFifo)
}

func TestLauncher_StopSequence_Lifo_StopError(t *testing.T) {
	runLauncherWithFailingStop(t, launcher.RunnerServicesSequenceLifo)
}

// --- Signals enabled (covers signal.Notify wiring), stopped via context ---

func TestLauncher_SignalsEnabled_StopViaContext(t *testing.T) {
	// Signals are enabled (default): this exercises the signal.Notify branch.
	// Shutdown is driven by ln.Stop() (context cancel), not an OS signal, so the
	// real-signal / force-exit branches are intentionally out of scope here.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	started := make(chan struct{})
	ln := launcher.New(
		launcher.WithLogger(quietExtended()),
		launcher.WithContext(ctx),
		launcher.WithAfterStart(func() error { close(started); return nil }),
	)
	ln.ServicesRunner().Register(
		launcher.NewService(
			launcher.WithServiceName("svc"),
			launcher.WithStart(blockingStart),
			launcher.WithStop(noopStop),
		),
	)

	errCh := make(chan error, 1)
	go func() { errCh <- ln.Run() }()

	select {
	case <-started:
	case err := <-errCh:
		t.Fatalf("Run returned before startup: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("service did not start in time")
	}

	ln.Stop()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
}

// --- Ops-enabled launcher run (real network, hermetic random port) ---

type opsHealthyService struct{}

func (opsHealthyService) Name() string                    { return "app-with-health" }
func (opsHealthyService) Start(ctx context.Context) error { <-ctx.Done(); return nil }
func (opsHealthyService) Stop(context.Context) error      { return nil }
func (opsHealthyService) Interval() time.Duration         { return time.Hour }
func (opsHealthyService) Healthy(context.Context) error   { return nil }

func TestLauncher_OpsEnabled_RegistersOpsServices(t *testing.T) {
	// Real network (ops http server on a random port), so no synctest here.
	// Synchronize via the AfterStart hook — it fires only once every service
	// (including the ops http listener + health worker) has started, giving a
	// race-free point to trigger shutdown without polling service state.
	started := make(chan struct{})

	ln := launcher.New(
		launcher.WithSignal(false),
		launcher.WithLogger(quietExtended()),
		launcher.WithAfterStart(func() error { close(started); return nil }),
		launcher.WithOpsConfig(ops.Config{
			Enabled: true,
			Network: "tcp",
			Healthy: ops.HealthCheckerConfig{
				Enabled:       true,
				Path:          "/healthy",
				Port:          "0", // random free port
				LivenessPath:  "/livez",
				ReadinessPath: "/readyz",
			},
		}),
	)

	// Registering a service exposing a HealthChecker exercises hcServices()
	// and stateProviders() wiring inside Run's ops-registration block.
	ln.ServicesRunner().Register(
		launcher.NewService(launcher.WithService(opsHealthyService{})),
	)

	errCh := make(chan error, 1)
	go func() { errCh <- ln.Run() }()

	select {
	case <-started:
	case err := <-errCh:
		t.Fatalf("Run returned before startup completed: %v", err)
	case <-time.After(10 * time.Second):
		ln.Stop()
		t.Fatal("services did not start in time")
	}

	ln.Stop()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
}
