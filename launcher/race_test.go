package launcher_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tkcrm/mx/launcher"
)

// State() is read from other goroutines (the ops /livez and /readyz probe
// handlers) while the service's own goroutine transitions its state. This must
// be data-race free. Run under -race; deliberately NOT a synctest test, because
// synctest serialises goroutines and would mask the very race being guarded.
func TestService_State_ConcurrentReads_NoRace(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	svc := launcher.NewService(
		launcher.WithServiceName("racy"),
		launcher.WithStart(blockingStart),
		launcher.WithStop(noopStop),
	)
	svc.Options().Context = ctx

	// Hammer State() from several readers throughout the whole lifecycle.
	stop := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = svc.State()
				}
			}
		}()
	}

	// Writer: the service goroutine transitions Starting → Running (and later
	// Stopping → Stopped) concurrently with the readers above.
	startErr := make(chan error, 1)
	go func() { startErr <- svc.Start() }()

	select {
	case <-svc.Ready(): // channel op — safe wait for Running without polling State()
	case <-time.After(5 * time.Second):
		close(stop)
		t.Fatal("service did not become ready in time")
	}

	cancel()
	if err := <-startErr; err != nil {
		close(stop)
		t.Fatalf("Start returned error: %v", err)
	}

	if err := svc.Stop(); err != nil { // Stopping → Stopped writes, readers still active
		close(stop)
		t.Fatalf("Stop returned error: %v", err)
	}

	close(stop)
	readers.Wait()

	if got := svc.State(); got != launcher.ServiceStateStopped {
		t.Fatalf("final state = %v; want stopped", got)
	}
}
