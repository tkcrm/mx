package launcher_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/tkcrm/mx/launcher"
)

// readinessService is a struct service that reports readiness via a channel the
// test controls, so tests can decide exactly when it becomes "operational".
type readinessService struct {
	name  string
	ready chan struct{}
}

func newReadinessService(name string) *readinessService {
	return &readinessService{name: name, ready: make(chan struct{})}
}

func (s *readinessService) Name() string                    { return s.name }
func (s *readinessService) Start(ctx context.Context) error { <-ctx.Done(); return nil }
func (s *readinessService) Stop(context.Context) error      { return nil }
func (s *readinessService) Ready() <-chan struct{}          { return s.ready }
func (s *readinessService) markReady()                      { close(s.ready) }

// A reporting service must stay Starting until it signals ready, then become
// Running — not the instant its Start goroutine is scheduled.
func TestService_Readiness_GatesRunningState(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		impl := newReadinessService("db")
		svc := launcher.NewService(launcher.WithService(impl))
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		// Not ready yet: must be Starting, and Ready() must not be closed.
		if svc.State() != launcher.ServiceStateStarting {
			t.Fatalf("before ready: state = %v; want starting", svc.State())
		}
		select {
		case <-svc.Ready():
			t.Fatal("Ready() closed before the service signalled readiness")
		default:
		}

		impl.markReady()
		synctest.Wait()

		if svc.State() != launcher.ServiceStateRunning {
			t.Fatalf("after ready: state = %v; want running", svc.State())
		}
		select {
		case <-svc.Ready():
		default:
			t.Fatal("Ready() not closed after the service signalled readiness")
		}

		cancel()
		synctest.Wait()
	})
}

// A service that becomes ready must NOT be killed by StartupTimeout, no matter
// how long it then runs. This is the core of bug #1.
func TestService_StartupTimeout_ReadyService_NotKilled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		impl := newReadinessService("server")
		svc := launcher.NewService(
			launcher.WithService(impl),
			launcher.WithStartupTimeout(5*time.Second),
		)
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()
		synctest.Wait()

		impl.markReady() // ready well within the timeout
		synctest.Wait()

		// Run far past the startup timeout — the service must stay healthy.
		time.Sleep(time.Hour)
		synctest.Wait()

		if svc.State() != launcher.ServiceStateRunning {
			t.Fatalf("state = %v; want running (ready service must not be killed by StartupTimeout)", svc.State())
		}

		cancel()
		synctest.Wait()
		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	})
}

// StartFn erroring before it signals ready must surface as a startup failure.
func TestService_Readiness_StartErrorBeforeReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		bootErr := errors.New("failed to connect")
		ready := make(chan struct{}) // never closed

		svc := launcher.NewService(
			launcher.WithServiceName("db"),
			launcher.WithStart(func(context.Context) error { return bootErr }),
			launcher.WithStop(noopStop),
			launcher.WithReadiness(ready),
		)
		svc.Options().Context = context.Background()

		if err := svc.Start(); !errors.Is(err, bootErr) {
			t.Fatalf("Start error = %v; want %v", err, bootErr)
		}
		if svc.State() != launcher.ServiceStateFailed {
			t.Fatalf("state = %v; want failed", svc.State())
		}
	})
}

// StartFn returning cleanly before it signals ready is a clean startup exit,
// and must NOT leave the service stuck in the Starting state (it ran to
// completion). It ends Running, consistent with a non-reporting service.
func TestService_Readiness_CleanExitBeforeReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ready := make(chan struct{}) // never closed

		svc := launcher.NewService(
			launcher.WithServiceName("oneshot"),
			launcher.WithStart(func(context.Context) error { return nil }),
			launcher.WithStop(noopStop),
			launcher.WithReadiness(ready),
		)
		svc.Options().Context = context.Background()

		if err := svc.Start(); err != nil {
			t.Fatalf("Start error = %v; want nil", err)
		}
		if got := svc.State(); got != launcher.ServiceStateRunning {
			t.Fatalf("state after clean exit before ready = %v; want running (must not stay starting)", got)
		}
	})
}

// A readiness-reporting service in a priority group that never becomes ready
// (and has no StartupTimeout) blocks the startup barrier — but Stop() (context
// cancel) must unblock Run rather than hang the launcher forever.
func TestLauncher_StartupPriority_NeverReady_ContextEscapes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := newReadinessService("db") // never marked ready

		ln := newTestLauncher()
		ln.ServicesRunner().Register(
			launcher.NewService(
				launcher.WithService(db),
				launcher.WithStartupPriority(1),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		// Startup is parked on the priority-1 db's readiness. Cancelling must
		// release it instead of hanging.
		ln.Stop()
		synctest.Wait()

		select {
		case err := <-errCh:
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Run error = %v; want context.Canceled", err)
			}
		default:
			t.Fatal("Run did not return after Stop() — startup wait hung")
		}
	})
}

// Cancelling the context while a reporting service is still starting up must
// shut it down cleanly (no startup-timeout error), even with no timeout set.
func TestService_Readiness_ContextCancelDuringStartup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		impl := newReadinessService("db") // never marked ready
		svc := launcher.NewService(launcher.WithService(impl))
		svc.Options().Context = ctx

		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start() }()
		synctest.Wait()

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start error = %v; want nil (cancelled during startup)", err)
		}
	})
}

// WithReadiness on an inline service must gate its state the same way the
// ReadinessReporter interface does.
func TestService_WithReadinessOption_Gates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ready := make(chan struct{})

		svc := launcher.NewService(
			launcher.WithServiceName("inline"),
			launcher.WithStart(blockingStart),
			launcher.WithStop(noopStop),
			launcher.WithReadiness(ready),
		)
		svc.Options().Context = ctx

		go func() { _ = svc.Start() }()
		synctest.Wait()

		if svc.State() != launcher.ServiceStateStarting {
			t.Fatalf("state = %v; want starting before readiness", svc.State())
		}

		close(ready)
		synctest.Wait()

		if svc.State() != launcher.ServiceStateRunning {
			t.Fatalf("state = %v; want running after readiness", svc.State())
		}

		cancel()
		synctest.Wait()
	})
}

// The startup-priority barrier must wait for a group's services to actually
// report ready before starting the next group. This is the core of bug #2:
// a priority-0 service must not start until the priority-1 database is ready.
func TestLauncher_StartupPriority_WaitsForReadiness(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := newReadinessService("db")
		var appStarted atomic.Bool

		ln := newTestLauncher()
		ln.ServicesRunner().Register(
			launcher.NewService(
				launcher.WithService(db),
				launcher.WithStartupPriority(1),
			),
			launcher.NewService(
				launcher.WithServiceName("app"),
				launcher.WithStart(func(ctx context.Context) error {
					appStarted.Store(true)
					<-ctx.Done()
					return nil
				}),
				launcher.WithStop(noopStop),
			),
		)

		errCh := make(chan error, 1)
		go func() { errCh <- ln.Run() }()
		synctest.Wait()

		// db has not signalled ready → the priority-0 app must not have started.
		if appStarted.Load() {
			t.Fatal("app (priority 0) started before the priority-1 db was ready")
		}
		if st := getState(t, ln, "db"); st != launcher.ServiceStateStarting {
			t.Fatalf("db state = %v; want starting", st)
		}

		db.markReady()
		// Let the priority barrier advance past the inter-group yield.
		time.Sleep(time.Millisecond)
		synctest.Wait()

		if !appStarted.Load() {
			t.Fatal("app did not start after the priority-1 db became ready")
		}

		ln.Stop()
		synctest.Wait()
		if err := <-errCh; err != nil {
			t.Fatalf("Run error: %v", err)
		}
	})
}

func getState(t *testing.T, ln launcher.ILauncher, name string) launcher.ServiceState {
	t.Helper()
	svc, ok := ln.ServicesRunner().Get(name)
	if !ok {
		t.Fatalf("service %q not registered", name)
	}
	return svc.State()
}
