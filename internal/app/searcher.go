package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	debug "github.com/tekig/clerk/internal/gateway/debug"
	sgrpc "github.com/tekig/clerk/internal/gateway/grpc"
	webui "github.com/tekig/clerk/internal/gateway/web-ui"
	"github.com/tekig/clerk/internal/repository"
	awss3 "github.com/tekig/clerk/internal/repository/aws-s3"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/searcher"
)

type SearcherConfig struct {
	Logger struct {
		Level  *string
		Format string
	}
	Storage struct {
		Type string
		S3   struct {
			Endpoint     string
			Bucket       string
			AccessKey    string
			AccessSecret string
		}
	}
	Cache struct {
		Type string
		Mem  *struct {
			MaxSize *int
		}
	}
	Recorder struct {
		Address []string
	}
	Gateway struct {
		GRPC *struct {
			Enabled bool
			Address string
		}
		WebUI *struct {
			Enabled bool
			Address string
		}
		Debug *struct {
			Enabled bool
			Address string
		}
	}
}

type Searcher struct {
	searcher *searcher.Searcher

	debug *debug.Debug
	grpc  *sgrpc.Searcher
	webui *webui.WebUI
}

func NewSearcher() (*Searcher, error) {
	var config SearcherConfig
	if err := readConfig(&config); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	switch config.Logger.Format {
	case "text", "":
		slog.SetDefault(
			slog.New(slog.NewTextHandler(os.Stdout, nil)),
		)
	case "json":
		slog.SetDefault(
			slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		)
	default:
		return nil, fmt.Errorf("unknow logger format `%s`", config.Logger.Format)
	}

	if config.Logger.Level != nil {
		var level slog.Level

		if err := level.UnmarshalText([]byte(*config.Logger.Level)); err != nil {
			return nil, fmt.Errorf("unmarshal log level `%s`: %w", *config.Logger.Level, err)
		}

		slog.SetLogLoggerLevel(level)
	}

	var storage repository.Storage
	switch config.Storage.Type {
	case "awss3":
		s, err := awss3.NewStorage(awss3.StorageConfig{
			Endpoint:     config.Storage.S3.Endpoint,
			Bucket:       config.Storage.S3.Bucket,
			AccessKey:    config.Storage.S3.AccessKey,
			AccessSecret: config.Storage.S3.AccessSecret,
		})
		if err != nil {
			return nil, fmt.Errorf("new storage: %w", err)
		}

		storage = s
	default:
		return nil, fmt.Errorf("unknown storage type `%s`", config.Storage.Type)
	}

	var cache repository.Cache
	switch config.Cache.Type {
	case "mem":
		var options []mem.OptionCache
		if config.Cache.Mem != nil && config.Cache.Mem.MaxSize != nil {
			options = append(options, mem.MaxSizeCache(*config.Cache.Mem.MaxSize))
		}

		cache = mem.NewCache(options...)
	default:
		return nil, fmt.Errorf("unknown cache type `%s`", config.Cache.Type)
	}

	s, err := searcher.NewSearcher(storage, cache, searcher.Readers(config.Recorder.Address))
	if err != nil {
		return nil, fmt.Errorf("searcher: %w", err)
	}

	var debugGateway *debug.Debug
	if config.Gateway.Debug != nil && config.Gateway.Debug.Enabled {
		g, err := debug.NewDebug(debug.DebugConfig{
			Address: config.Gateway.Debug.Address,
		})
		if err != nil {
			return nil, fmt.Errorf("http gateway: %w", err)
		}

		debugGateway = g
	}

	var grpcGateway *sgrpc.Searcher
	if config.Gateway.Debug != nil && config.Gateway.Debug.Enabled {
		g, err := sgrpc.NewSearcher(sgrpc.SearcherConfig{
			Searcher: s,
			Address:  config.Gateway.GRPC.Address,
		})
		if err != nil {
			return nil, fmt.Errorf("http gateway: %w", err)
		}

		grpcGateway = g
	}

	var webuiGateway *webui.WebUI
	if config.Gateway.WebUI != nil && config.Gateway.WebUI.Enabled {
		g, err := webui.NewWebUI(webui.SearcherConfig{
			Searcher: s,
			Address:  config.Gateway.WebUI.Address,
		})
		if err != nil {
			return nil, fmt.Errorf("webui gateway: %w", err)
		}

		webuiGateway = g
	}

	return &Searcher{
		searcher: s,
		debug:    debugGateway,
		grpc:     grpcGateway,
		webui:    webuiGateway,
	}, nil
}

func (r *Searcher) Run() (runErr error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	once := sync.Once{}

	if r.grpc != nil {
		go func() {
			defer cancel()

			if err := r.grpc.Run(); err != nil {
				once.Do(func() {
					runErr = fmt.Errorf("run grpc: %w", err)
				})
			}
		}()
	}

	if r.debug != nil {
		go func() {
			defer cancel()

			if err := r.debug.Run(); err != nil {
				once.Do(func() {
					runErr = fmt.Errorf("run http: %w", err)
				})
			}
		}()
	}

	if r.webui != nil {
		go func() {
			defer cancel()

			if err := r.webui.Run(); err != nil {
				once.Do(func() {
					runErr = fmt.Errorf("run webui: %w", err)
				})
			}
		}()
	}

	<-ctx.Done()

	return runErr
}

func (r *Searcher) Shutdown() error {
	if r.debug != nil {
		if err := r.debug.Shutdown(); err != nil {
			return fmt.Errorf("shutdown http: %w", err)
		}
	}

	if r.grpc != nil {
		if err := r.grpc.Shutdown(); err != nil {
			return fmt.Errorf("shutdown grpc: %w", err)
		}
	}

	return nil
}
