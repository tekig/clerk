package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	debug "github.com/tekig/clerk/internal/gateway/debug"
	sgrpc "github.com/tekig/clerk/internal/gateway/grpc"
	otelproxy "github.com/tekig/clerk/internal/otel-proxy"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/repository"
	awss3 "github.com/tekig/clerk/internal/repository/aws-s3"
	rgrpc "github.com/tekig/clerk/internal/repository/grpc"
)

type RecorderConfig struct {
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
	Searcher struct {
		Address string
	}
	Recorder struct {
		BlockSize *string
		ChunkSize *string
		BlocksDir *string
	}
	OTELProxy struct {
		Target          string
		DefaultStrategy string
		FormatURL       string
		Rules           []otelproxy.ConfigRule
	}
	Gateway struct {
		GRPC *struct {
			Enabled bool
			Address string
			Gateway string
		}
		Debug *struct {
			Enabled bool
			Address string
		}
	}
}

type Recorder struct {
	recorder *recorder.Recorder

	debug *debug.Debug
	grpc  *sgrpc.Recorder

	searcher repository.Searcher
}

func NewRecorder() (*Recorder, error) {
	var config RecorderConfig
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

	searcher, err := rgrpc.NewSearcher(config.Searcher.Address)
	if err != nil {
		return nil, fmt.Errorf("new searcher: %w", err)
	}

	var recorderOptions []recorder.Option
	if config.Recorder.BlockSize != nil {
		size, err := toBytes(*config.Recorder.BlockSize)
		if err != nil {
			return nil, fmt.Errorf("to bytes block size: %w", err)
		}

		recorderOptions = append(recorderOptions, recorder.MaxBlockSize(size))
	}
	if config.Recorder.BlocksDir != nil {
		if _, err := os.ReadDir(*config.Recorder.BlocksDir); err != nil {
			return nil, fmt.Errorf("read blocks dir: %w", err)
		}

		recorderOptions = append(recorderOptions, recorder.BlocksDir(*config.Recorder.BlocksDir))
	}
	if config.Recorder.ChunkSize != nil {
		size, err := toBytes(*config.Recorder.ChunkSize)
		if err != nil {
			return nil, fmt.Errorf("to bytes chunk size: %w", err)
		}

		recorderOptions = append(recorderOptions, recorder.MaxChunkSize(size))
	}

	r, err := recorder.NewRecorder(storage, searcher, recorderOptions...)
	if err != nil {
		return nil, fmt.Errorf("new recorder: %w", err)
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

	proxy, err := otelproxy.New(otelproxy.Config{
		Target:          config.OTELProxy.Target,
		Recorder:        r,
		FormatURL:       config.OTELProxy.FormatURL,
		DefaultStrategy: config.OTELProxy.DefaultStrategy,
		Rules:           config.OTELProxy.Rules,
	})
	if err != nil {
		return nil, fmt.Errorf("otel proxy: %w", err)
	}

	var grpcGateway *sgrpc.Recorder
	if config.Gateway.Debug != nil && config.Gateway.Debug.Enabled {
		g, err := sgrpc.NewRecorder(sgrpc.RecorderConfig{
			Recorder:    r,
			OTELProxy:   proxy,
			GRPCAddress: config.Gateway.GRPC.Address,
			HTTPAddress: config.Gateway.GRPC.Gateway,
		})
		if err != nil {
			return nil, fmt.Errorf("http gateway: %w", err)
		}

		grpcGateway = g
	}

	return &Recorder{
		recorder: r,
		debug:    debugGateway,
		grpc:     grpcGateway,
		searcher: searcher,
	}, nil
}

func (r *Recorder) Run() (runErr error) {
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

	<-ctx.Done()

	return runErr
}

func (r *Recorder) Shutdown() error {
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

	if err := r.recorder.Shutdown(); err != nil {
		return fmt.Errorf("shutdown recorder: %w", err)
	}

	if err := r.searcher.Close(); err != nil {
		return fmt.Errorf("close searcher: %w", err)
	}

	return nil
}
