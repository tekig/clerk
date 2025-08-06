package http

import (
	_ "net/http/pprof"

	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Debug struct {
	server *http.Server
}

type DebugConfig struct {
	Address string
}

func NewDebug(config DebugConfig) (*Debug, error) {
	g := &Debug{
		server: &http.Server{
			Addr:    config.Address,
			Handler: http.DefaultServeMux,
		},
	}

	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("exporter prom: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(provider)

	http.Handle("/metrics", promhttp.Handler())

	return g, nil
}

func (g *Debug) Run() error {
	if err := g.server.ListenAndServe(); err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	return nil
}

func (g *Debug) Shutdown() error {
	return g.server.Shutdown(context.Background())
}
