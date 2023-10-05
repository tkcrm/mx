package ops

import (
	"net/http"
	"net/http/pprof"
)

type profilerOpsService struct {
	config ProfilerConfig
}

func newProfilerOpsService(cfg ProfilerConfig) *profilerOpsService {
	return &profilerOpsService{cfg}
}

type ProfilerConfig struct {
	Enabled bool   `default:"false" usage:"allows to enable profiler"`
	Path    string `default:"/debug/pprof" usage:"allows to set custom profiler path"`
	Port    string `default:"10000" usage:"allows to set custom profiler port"`
}

func (s profilerOpsService) Name() string { return "profiler" }

func (s profilerOpsService) getEnabled() bool { return s.config.Enabled }

func (s profilerOpsService) getPort() string { return s.config.Port }

func (s profilerOpsService) initService(mux *http.ServeMux) {
	mux.HandleFunc(s.config.Path+"/", pprof.Index)
	mux.HandleFunc(s.config.Path+"/cmdline", pprof.Cmdline)
	mux.HandleFunc(s.config.Path+"/profile", pprof.Profile)
	mux.HandleFunc(s.config.Path+"/symbol", pprof.Symbol)
	mux.HandleFunc(s.config.Path+"/trace", pprof.Trace)
}

var _ opsService = (*profilerOpsService)(nil)
