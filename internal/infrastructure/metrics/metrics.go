package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(
			collectors.ProcessCollectorOpts{},
		),
		collectors.NewBuildInfoCollector(),
	)

	return registry
}

func NewHandler(gatherer prometheus.Gatherer) http.Handler {
	return promhttp.HandlerFor(
		gatherer,
		promhttp.HandlerOpts{},
	)
}
