package prometheus

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tkcrm/mx/logger"

	_ "github.com/mitchellh/mapstructure"
)

type Server struct {
	logger logger.Logger
	config Config

	srv *http.Server

	requestCounter *prometheus.CounterVec
}

func New(logger logger.Logger, config Config, serviceName string) *Server {
	if serviceName == "" {
		logger.Errorf("prometheus error: service name is empty")
		serviceName = "unknown-service-name"
	}

	return &Server{
		logger: logger,
		config: config,
		// TODO: make it configurable, like hc
		requestCounter: promauto.With(prometheus.NewRegistry()).NewCounterVec(
			prometheus.CounterOpts{
				Namespace: strings.ReplaceAll(serviceName, "-", "_"),
				Name:      "requests_counter",
				Help:      "",
			}, []string{"query", "status"}),
	}
}

func (s *Server) Name() string { return "prometheus" }

// Start prometheus server
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}

	r := mux.NewRouter()
	r.Path(s.config.Endpoint).Handler(promhttp.Handler())

	s.srv = &http.Server{Addr: ":" + s.config.Port, Handler: r}

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start prometheus on port %s: %v", s.config.Port, err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) Enabled() bool {
	return s.config.Enabled
}

func (s *Server) IncrementRequestsCount(query, result string) {
	s.requestCounter.WithLabelValues(query, result).Inc()
}
