package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/tkcrm/mx/mxtypes"
)

func TestNewHealthCheckerOpsService_Defaults(t *testing.T) {
	// Empty paths must fall back to their defaults.
	svc := newHealthCheckerOpsService(quietLog(), HealthCheckerConfig{})
	if svc.config.Path != "/healthy" {
		t.Errorf("Path = %q; want /healthy", svc.config.Path)
	}
	if svc.config.LivenessPath != "/livez" {
		t.Errorf("LivenessPath = %q; want /livez", svc.config.LivenessPath)
	}
	if svc.config.ReadinessPath != "/readyz" {
		t.Errorf("ReadinessPath = %q; want /readyz", svc.config.ReadinessPath)
	}

	// Explicit paths must be preserved.
	custom := newHealthCheckerOpsService(quietLog(), HealthCheckerConfig{
		Path:          "/h",
		LivenessPath:  "/l",
		ReadinessPath: "/r",
	})
	if custom.config.Path != "/h" || custom.config.LivenessPath != "/l" || custom.config.ReadinessPath != "/r" {
		t.Errorf("custom paths not preserved: %+v", custom.config)
	}
}

func TestHealthChecker_Getters(t *testing.T) {
	svc := newHealthCheckerOpsService(quietLog(), HealthCheckerConfig{
		Enabled: true,
		Port:    "12345",
	})
	if svc.Name() != "ops-health-checker" {
		t.Errorf("Name = %q", svc.Name())
	}
	if !svc.getEnabled() {
		t.Error("getEnabled = false; want true")
	}
	if svc.getPort() != "12345" {
		t.Errorf("getPort = %q; want 12345", svc.getPort())
	}
	if got := svc.getHTTPOptions(); len(got) != 0 {
		t.Errorf("getHTTPOptions len = %d; want 0", len(got))
	}
	if err := svc.Stop(context.Background()); err != nil {
		t.Errorf("Stop error: %v", err)
	}
}

func TestHealthChecker_ServeHTTP_StatusCodes(t *testing.T) {
	tests := []struct {
		name     string
		codes    map[string]HealthCheckCode
		wantCode int
	}{
		{"all ok", map[string]HealthCheckCode{"a": HealthCheckCodeOk, "b": HealthCheckCodeOk}, http.StatusOK},
		{"one starting", map[string]HealthCheckCode{"a": HealthCheckCodeOk, "b": HealthCheckCodeServiceStarting}, http.StatusFailedDependency},
		{"error wins over starting", map[string]HealthCheckCode{"a": HealthCheckCodeServiceStarting, "b": HealthCheckCodeError}, http.StatusServiceUnavailable},
		{"empty", map[string]HealthCheckCode{}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newHealthCheckerOpsService(quietLog(), HealthCheckerConfig{})
			for k, v := range tt.codes {
				svc.resp.Store(k, v)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/healthy", nil)
			svc.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantCode)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q; want application/json", ct)
			}

			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON body: %v", err)
			}
			if len(body) != len(tt.codes) {
				t.Errorf("body has %d entries; want %d", len(body), len(tt.codes))
			}
		})
	}
}

func TestHealthChecker_ServeLiveness(t *testing.T) {
	tests := []struct {
		name     string
		states   []mxtypes.ServiceState
		wantCode int
		wantStat string
	}{
		{"all running", []mxtypes.ServiceState{mxtypes.ServiceStateRunning, mxtypes.ServiceStateRunning}, http.StatusOK, "ok"},
		{"one failed", []mxtypes.ServiceState{mxtypes.ServiceStateRunning, mxtypes.ServiceStateFailed}, http.StatusServiceUnavailable, "failed"},
		{"starting is alive", []mxtypes.ServiceState{mxtypes.ServiceStateStarting}, http.StatusOK, "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HealthCheckerConfig{}
			providers := make([]mxtypes.StateProvider, len(tt.states))
			for i, st := range tt.states {
				providers[i] = fakeStateProvider{name: "svc" + string(rune('a'+i)), state: st}
			}
			cfg.AddStateList(providers)

			svc := newHealthCheckerOpsService(quietLog(), cfg)

			rec := httptest.NewRecorder()
			svc.serveLiveness(rec, httptest.NewRequest(http.MethodGet, "/livez", nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantCode)
			}
			var body struct {
				Status   string            `json:"status"`
				Services map[string]string `json:"services"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if body.Status != tt.wantStat {
				t.Errorf("status = %q; want %q", body.Status, tt.wantStat)
			}
			if len(body.Services) != len(tt.states) {
				t.Errorf("services len = %d; want %d", len(body.Services), len(tt.states))
			}
		})
	}
}

func TestHealthChecker_ServeReadiness(t *testing.T) {
	tests := []struct {
		name       string
		states     map[string]mxtypes.ServiceState
		healthResp map[string]HealthCheckCode
		wantCode   int
		wantStat   string
	}{
		{
			name:     "all running no health",
			states:   map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateRunning},
			wantCode: http.StatusOK,
			wantStat: "ok",
		},
		{
			name:     "starting state",
			states:   map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateStarting},
			wantCode: http.StatusFailedDependency,
			wantStat: "starting",
		},
		{
			name:     "idle state",
			states:   map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateIdle},
			wantCode: http.StatusFailedDependency,
			wantStat: "starting",
		},
		{
			name:     "failed state",
			states:   map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateFailed},
			wantCode: http.StatusServiceUnavailable,
			wantStat: "unavailable",
		},
		{
			name:       "running but health error",
			states:     map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateRunning},
			healthResp: map[string]HealthCheckCode{"a": HealthCheckCodeError},
			wantCode:   http.StatusServiceUnavailable,
			wantStat:   "unavailable",
		},
		{
			name:       "running but health starting",
			states:     map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateRunning},
			healthResp: map[string]HealthCheckCode{"a": HealthCheckCodeServiceStarting},
			wantCode:   http.StatusFailedDependency,
			wantStat:   "starting",
		},
		{
			name:       "running with health ok",
			states:     map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateRunning},
			healthResp: map[string]HealthCheckCode{"a": HealthCheckCodeOk},
			wantCode:   http.StatusOK,
			wantStat:   "ok",
		},
		{
			name:       "health for unknown service",
			states:     map[string]mxtypes.ServiceState{"a": mxtypes.ServiceStateRunning},
			healthResp: map[string]HealthCheckCode{"ghost": HealthCheckCodeOk},
			wantCode:   http.StatusOK,
			wantStat:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HealthCheckerConfig{}
			providers := make([]mxtypes.StateProvider, 0, len(tt.states))
			for name, st := range tt.states {
				providers = append(providers, fakeStateProvider{name: name, state: st})
			}
			cfg.AddStateList(providers)

			svc := newHealthCheckerOpsService(quietLog(), cfg)
			for name, code := range tt.healthResp {
				svc.resp.Store(name, code)
			}

			rec := httptest.NewRecorder()
			svc.serveReadiness(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d; want %d", rec.Code, tt.wantCode)
			}
			var body struct {
				Status   string                           `json:"status"`
				Services map[string]readinessServiceEntry `json:"services"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if body.Status != tt.wantStat {
				t.Errorf("status = %q; want %q", body.Status, tt.wantStat)
			}
		})
	}
}

func TestHealthChecker_InitService_Routes(t *testing.T) {
	cfg := HealthCheckerConfig{Path: "/healthy"}
	cfg.AddStateList([]mxtypes.StateProvider{fakeStateProvider{name: "a", state: mxtypes.ServiceStateRunning}})
	svc := newHealthCheckerOpsService(quietLog(), cfg)

	mux := http.NewServeMux()
	svc.initService(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	for _, path := range []string{"/healthy", "/livez", "/readyz"} {
		resp, err := ts.Client().Get(ts.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d; want 200", path, resp.StatusCode)
		}
	}
}

func TestHealthChecker_InitService_ProbesDisabled(t *testing.T) {
	// Empty liveness/readiness paths must not register those routes.
	// newHealthCheckerOpsService defaults empty paths, so build the struct directly.
	svc := &healthCheckerOpsService{
		log:    quietLog(),
		config: HealthCheckerConfig{Path: "/healthy"},
		resp:   new(sync.Map),
	}
	mux := http.NewServeMux()
	svc.initService(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	// /healthy is registered
	resp, err := ts.Client().Get(ts.URL + "/healthy")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthy status = %d; want 200", resp.StatusCode)
	}

	// /livez is NOT registered → default mux returns 404
	resp, err = ts.Client().Get(ts.URL + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("/livez status = %d; want 404 (disabled)", resp.StatusCode)
	}
}

func TestHealthChecker_Start_PollingTransitions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var dbCalls atomic.Int32

		// db: error on the first poll, healthy after → error → recovery path.
		db := &fakeHealthChecker{
			name:     "db",
			interval: time.Second,
			healthy: func(_ context.Context) error {
				if dbCalls.Add(1) == 1 {
					return ErrHealthCheckError
				}
				return nil
			},
		}
		// queue: always reports "starting" → exercises the starting branch.
		queue := &fakeHealthChecker{
			name:     "queue",
			interval: time.Second,
			healthy:  func(_ context.Context) error { return ErrHealthCheckServiceStarting },
		}

		cfg := HealthCheckerConfig{}
		// nil entry must be tolerated (wg.Done + continue).
		cfg.AddServicesList([]mxtypes.HealthChecker{db, queue, nil})

		svc := newHealthCheckerOpsService(quietLog(), cfg)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start(ctx) }()

		// The first poll runs immediately at startup (no interval wait).
		synctest.Wait()

		if got := dbCalls.Load(); got != 1 {
			t.Fatalf("db polled %d times immediately; want 1 (immediate first poll)", got)
		}
		if got := loadCode(t, svc, "db"); got != HealthCheckCodeError {
			t.Errorf("db after first poll = %v; want error", got)
		}
		if got := loadCode(t, svc, "queue"); got != HealthCheckCodeServiceStarting {
			t.Errorf("queue after first poll = %v; want starting", got)
		}

		// Second poll one interval later: db recovers → ok (recovery path).
		time.Sleep(time.Second)
		synctest.Wait()

		if got := loadCode(t, svc, "db"); got != HealthCheckCodeOk {
			t.Errorf("db after second poll = %v; want ok", got)
		}

		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	})
}

// A service whose Healthy always succeeds (e.g. ping-pong) must be reported OK
// as soon as the worker starts — not "starting" for a whole interval. This is
// the regression the flapping /readyz + "fixed" log came from.
func TestHealthChecker_HealthyService_ReadyImmediately(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		pingpong := &fakeHealthChecker{
			name:     "ping-pong",
			interval: time.Second,
			healthy: func(_ context.Context) error {
				calls.Add(1)
				return nil // no external dependencies — always healthy
			},
		}

		cfg := HealthCheckerConfig{}
		cfg.AddServicesList([]mxtypes.HealthChecker{pingpong})
		svc := newHealthCheckerOpsService(quietLog(), cfg)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start(ctx) }()

		// No time advance: the immediate first poll must already mark it OK,
		// with no "starting" window.
		synctest.Wait()

		if got := calls.Load(); got != 1 {
			t.Fatalf("healthy service polled %d times; want 1 immediate poll", got)
		}
		if got := loadCode(t, svc, "ping-pong"); got != HealthCheckCodeOk {
			t.Fatalf("ping-pong health = %v; want ok immediately (no starting window)", got)
		}

		cancel()
		synctest.Wait()
		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	})
}

func TestHealthChecker_Start_NoCheckers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc := newHealthCheckerOpsService(quietLog(), HealthCheckerConfig{})

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() { errCh <- svc.Start(ctx) }()

		synctest.Wait()
		cancel()
		synctest.Wait()

		if err := <-errCh; err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	})
}

// errResponseWriter fails every Write, to exercise JSON encode-error branches.
type errResponseWriter struct{ header http.Header }

func (w *errResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *errResponseWriter) Write([]byte) (int, error) { return 0, errTestWrite }
func (w *errResponseWriter) WriteHeader(int)           {}

func TestHealthChecker_ServeReadiness_SkipsMalformedRespEntries(t *testing.T) {
	cfg := HealthCheckerConfig{}
	cfg.AddStateList([]mxtypes.StateProvider{fakeStateProvider{name: "a", state: mxtypes.ServiceStateRunning}})
	svc := newHealthCheckerOpsService(quietLog(), cfg)

	// Defensive branches: non-string key and non-HealthCheckCode value are ignored.
	svc.resp.Store(42, HealthCheckCodeOk)
	svc.resp.Store("weird", "not-a-code")
	svc.resp.Store("a", HealthCheckCodeOk)

	rec := httptest.NewRecorder()
	svc.serveReadiness(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
	var body struct {
		Services map[string]readinessServiceEntry `json:"services"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := body.Services["weird"]; ok {
		t.Error("malformed value entry should not appear in output")
	}
	if body.Services["a"].Health != "ok" {
		t.Errorf("service a health = %q; want ok", body.Services["a"].Health)
	}
}

func TestHealthChecker_EncodeErrors(t *testing.T) {
	cfg := HealthCheckerConfig{}
	cfg.AddStateList([]mxtypes.StateProvider{fakeStateProvider{name: "a", state: mxtypes.ServiceStateRunning}})
	svc := newHealthCheckerOpsService(quietLog(), cfg)
	svc.resp.Store("a", HealthCheckCodeOk)

	// Each handler must not panic when the response writer fails.
	svc.ServeHTTP(&errResponseWriter{}, httptest.NewRequest(http.MethodGet, "/healthy", nil))
	svc.serveLiveness(&errResponseWriter{}, httptest.NewRequest(http.MethodGet, "/livez", nil))
	svc.serveReadiness(&errResponseWriter{}, httptest.NewRequest(http.MethodGet, "/readyz", nil))
}

func loadCode(t *testing.T, svc *healthCheckerOpsService, name string) HealthCheckCode {
	t.Helper()
	v, ok := svc.resp.Load(name)
	if !ok {
		t.Fatalf("no health code stored for %q", name)
	}
	code, ok := v.(HealthCheckCode)
	if !ok {
		t.Fatalf("stored value for %q is not a HealthCheckCode: %T", name, v)
	}
	return code
}
