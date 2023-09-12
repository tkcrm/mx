package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

// health implements service.Service
// and used as worker pool for HealthChecker.
type healthChecker struct {
	log logger.ExtendedLogger

	resp *sync.Map
	list []service.HealthChecker
}

// Name returns name of http server.
func (s healthChecker) Name() string { return "ops-health-checker" }

func newHealthChecker(log logger.ExtendedLogger, list ...service.HealthChecker) *healthChecker {
	wrk := &healthChecker{log: log, list: list, resp: new(sync.Map)}
	return wrk
}

// ServeHTTP implementation of http.Handler for OPS worker.
func (o *healthChecker) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	out := make(map[string]interface{})
	o.resp.Range(func(key, val any) bool {
		if name, ok := key.(string); ok {
			out[name] = val
		}

		return true
	})

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(out); err != nil {
		o.log.Errorf("could not write response: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// implementation of service.IService for OPS worker.
func (o *healthChecker) Start(ctx context.Context) error {
	wg := new(sync.WaitGroup)

	wg.Add(len(o.list))
	for i := 0; i < len(o.list); i++ {
		if o.list[i] == nil {
			wg.Done()
			continue
		}

		o.resp.Store(o.list[i].Name(), 0)

		// run health checker for each service
		go func(checker service.HealthChecker) {
			defer wg.Done()

			name := checker.Name()
			delay := checker.Interval()

			ticker := time.NewTimer(delay)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := checker.Healthy(ctx); err != nil {
						o.resp.Store(name, 1)
						o.log.Warnf("check service %s failed with error: %s", name, err)
					} else {
						o.resp.Store(name, 0)
					}

					ticker.Reset(delay)
				}
			}
		}(o.list[i])
	}

	<-ctx.Done()

	wg.Wait()

	return nil
}

func (o *healthChecker) Stop(ctx context.Context) error {
	return nil
}
