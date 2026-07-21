package ops

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkcrm/mx/transport/http_transport"
)

func TestOpsNew_AllDisabled_ReturnsEmpty(t *testing.T) {
	svcs := New(quietLog(), Config{})
	if len(svcs) != 0 {
		t.Fatalf("New with all disabled returned %d services; want 0", len(svcs))
	}
}

func TestOpsNew_MetricsOnly(t *testing.T) {
	svcs := New(quietLog(), Config{
		Enabled: true,
		Network: "tcp",
		Metrics: MetricsConfig{Enabled: true, Path: "/metrics", Port: "10000"},
	})
	// metrics → single http server.
	if len(svcs) != 1 {
		t.Fatalf("got %d services; want 1", len(svcs))
	}
	srv, ok := svcs[0].(*http_transport.HTTPServer)
	if !ok {
		t.Fatalf("service is %T; want *http_transport.HTTPServer", svcs[0])
	}
	if !srv.Enabled() {
		t.Error("ops http server should be enabled")
	}
}

func TestOpsNew_HealthyOnly(t *testing.T) {
	svcs := New(quietLog(), Config{
		Enabled: true,
		Network: "tcp",
		Healthy: HealthCheckerConfig{Enabled: true, Path: "/healthy", Port: "10000"},
	})
	// healthy → worker service + http server.
	if len(svcs) != 2 {
		t.Fatalf("got %d services; want 2", len(svcs))
	}
	names := map[string]bool{}
	for _, s := range svcs {
		names[s.Name()] = true
	}
	if !names["ops-health-checker"] {
		t.Errorf("missing health-checker worker; got %v", names)
	}
}

func TestOpsNew_ProfilerOnly(t *testing.T) {
	svcs := New(quietLog(), Config{
		Enabled:  true,
		Network:  "tcp",
		Profiler: ProfilerConfig{Enabled: true, Path: "/debug/pprof", Port: "10000"},
	})
	if len(svcs) != 1 {
		t.Fatalf("got %d services; want 1", len(svcs))
	}
}

func TestOpsNew_AllSamePort_SingleServer(t *testing.T) {
	svcs := New(quietLog(), Config{
		Enabled:  true,
		Network:  "tcp",
		Metrics:  MetricsConfig{Enabled: true, Path: "/metrics", Port: "10000"},
		Healthy:  HealthCheckerConfig{Enabled: true, Path: "/healthy", Port: "10000"},
		Profiler: ProfilerConfig{Enabled: true, Path: "/debug/pprof", Port: "10000"},
	})
	// One shared server on :10000 + health worker = 2 services.
	if len(svcs) != 2 {
		t.Fatalf("got %d services; want 2 (shared server + health worker)", len(svcs))
	}
	var servers int
	for _, s := range svcs {
		if _, ok := s.(*http_transport.HTTPServer); ok {
			servers++
		}
	}
	if servers != 1 {
		t.Fatalf("got %d http servers; want 1 (shared port)", servers)
	}
}

func TestOpsNew_DifferentPorts_MultipleServers(t *testing.T) {
	svcs := New(quietLog(), Config{
		Enabled:  true,
		Network:  "tcp",
		Metrics:  MetricsConfig{Enabled: true, Path: "/metrics", Port: "10001"},
		Profiler: ProfilerConfig{Enabled: true, Path: "/debug/pprof", Port: "10002"},
	})
	var servers int
	for _, s := range svcs {
		if _, ok := s.(*http_transport.HTTPServer); ok {
			servers++
		}
	}
	if servers != 2 {
		t.Fatalf("got %d http servers; want 2 (distinct ports)", servers)
	}
}

func TestOpsNew_ProfilerWithWriteTimeout_AppendsHTTPOptions(t *testing.T) {
	// Profiler with WriteTimeout>0 contributes extra http options to its server,
	// exercising the httpOpts append branch in New.
	svcs := New(quietLog(), Config{
		Enabled:  true,
		Network:  "tcp",
		Profiler: ProfilerConfig{Enabled: true, Path: "/debug/pprof", Port: "10000", WriteTimeout: 30},
	})
	var srv *http_transport.HTTPServer
	for _, s := range svcs {
		if hs, ok := s.(*http_transport.HTTPServer); ok {
			srv = hs
		}
	}
	if srv == nil {
		t.Fatal("no http server returned")
	}
	if srv.Config.WriteTimeout != 30 {
		t.Errorf("WriteTimeout = %d; want 30 (applied from profiler http option)", srv.Config.WriteTimeout)
	}
}

func TestConfig_GetHTTPOptionForPort(t *testing.T) {
	cfg := Config{Enabled: true, Network: "tcp", TracingEnabled: true}
	opt := cfg.getHTTPOptionForPort("9999")

	// Apply the option to a real server and check the resulting config.
	srv := http_transport.NewServer(opt)
	if srv.Config.Address != ":9999" {
		t.Errorf("Address = %q; want :9999", srv.Config.Address)
	}
	if srv.Config.Network != "tcp" {
		t.Errorf("Network = %q; want tcp", srv.Config.Network)
	}
	if srv.Config.NoTrace {
		t.Error("NoTrace = true; want false (tracing enabled)")
	}

	// Tracing disabled → NoTrace true.
	cfg.TracingEnabled = false
	srv2 := http_transport.NewServer(cfg.getHTTPOptionForPort("9999"))
	if !srv2.Config.NoTrace {
		t.Error("NoTrace = false; want true (tracing disabled)")
	}
}

// --- metrics service ---

func TestMetricsOpsService_Getters(t *testing.T) {
	svc := newMetricsOpsService(MetricsConfig{Enabled: true, Path: "/metrics", Port: "10000"})
	if svc.Name() != "metrics" {
		t.Errorf("Name = %q; want metrics", svc.Name())
	}
	if !svc.getEnabled() {
		t.Error("getEnabled = false")
	}
	if svc.getPort() != "10000" {
		t.Errorf("getPort = %q", svc.getPort())
	}
	if got := svc.getHTTPOptions(); len(got) != 0 {
		t.Errorf("getHTTPOptions len = %d; want 0", len(got))
	}
}

func TestMetricsOpsService_ServesMetrics(t *testing.T) {
	svc := newMetricsOpsService(MetricsConfig{Enabled: true, Path: "/metrics", Port: "10000"})
	mux := http.NewServeMux()
	svc.initService(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	resp, err := ts.Client().Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/metrics status = %d; want 200", resp.StatusCode)
	}
}

func TestMetricsOpsService_BasicAuth(t *testing.T) {
	svc := newMetricsOpsService(MetricsConfig{
		Enabled:   true,
		Path:      "/metrics",
		Port:      "10000",
		BasicAuth: http_transport.BasicAuthConfig{Enabled: true, Username: "user", Password: "pass"},
	})
	mux := http.NewServeMux()
	svc.initService(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	// No credentials → 401.
	resp, err := ts.Client().Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no-auth status = %d; want 401", resp.StatusCode)
	}

	// Correct credentials → 200.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/metrics", nil)
	req.SetBasicAuth("user", "pass")
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("auth status = %d; want 200", resp.StatusCode)
	}
}

// --- profiler service ---

func TestProfilerOpsService_Getters(t *testing.T) {
	svc := newProfilerOpsService(ProfilerConfig{Enabled: true, Path: "/debug/pprof", Port: "10000"})
	if svc.Name() != "profiler" {
		t.Errorf("Name = %q; want profiler", svc.Name())
	}
	if !svc.getEnabled() {
		t.Error("getEnabled = false")
	}
	if svc.getPort() != "10000" {
		t.Errorf("getPort = %q", svc.getPort())
	}
}

func TestProfilerOpsService_HTTPOptions_WriteTimeout(t *testing.T) {
	// WriteTimeout > 0 → one option.
	with := newProfilerOpsService(ProfilerConfig{WriteTimeout: 60})
	if got := with.getHTTPOptions(); len(got) != 1 {
		t.Errorf("with WriteTimeout: options len = %d; want 1", len(got))
	}
	// WriteTimeout == 0 → no options.
	without := newProfilerOpsService(ProfilerConfig{})
	if got := without.getHTTPOptions(); len(got) != 0 {
		t.Errorf("without WriteTimeout: options len = %d; want 0", len(got))
	}
}

func TestProfilerOpsService_ServesPprof(t *testing.T) {
	svc := newProfilerOpsService(ProfilerConfig{Enabled: true, Path: "/debug/pprof", Port: "10000"})
	mux := http.NewServeMux()
	svc.initService(mux)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	// index and a named handler.
	for _, path := range []string{"/debug/pprof/", "/debug/pprof/cmdline", "/debug/pprof/symbol"} {
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
