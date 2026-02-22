package http

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"

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
	http.HandleFunc("/debug/profiling", g.hdlrProfiling)

	return g, nil
}

func (g *Debug) hdlrProfiling(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if v := q.Get("mutex"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid mutex value: %s", err.Error()), http.StatusBadRequest)
			return
		}

		runtime.SetMutexProfileFraction(n)
		fmt.Fprintln(w, "mutex updated")
	}

	if v := q.Get("block"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid block value: %s", err.Error()), http.StatusBadRequest)
			return
		}

		runtime.SetBlockProfileRate(n)
		fmt.Fprintln(w, "block updated")
	}
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
