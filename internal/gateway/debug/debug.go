package http

import (
	_ "net/http/pprof"

	"context"
	"fmt"
	"net/http"
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
