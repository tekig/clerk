package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/tekig/clerk/internal/logger"
	otelproxy "github.com/tekig/clerk/internal/otel-proxy"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/uuid"
	otelcollector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

var _ pb.RecorderServer = (*Recorder)(nil)

type Recorder struct {
	recorder    *recorder.Recorder
	otelProxy   *otelproxy.Proxy
	grpcServer  *grpc.Server
	grpcAddress string
	pb.UnimplementedRecorderServer
	otelcollector.UnimplementedTraceServiceServer
}

type RecorderConfig struct {
	Recorder    *recorder.Recorder
	OTELProxy   *otelproxy.Proxy
	GRPCAddress string
}

func NewRecorder(config RecorderConfig) (*Recorder, error) {
	g := &Recorder{
		otelProxy: config.OTELProxy,
		recorder:  config.Recorder,
		grpcServer: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				logger.UnaryServerInterceptor(),
			),
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     2 * time.Minute,
				MaxConnectionAge:      5 * time.Minute,
				MaxConnectionAgeGrace: 2 * time.Minute,
				Time:                  5 * time.Minute,
				Timeout:               20 * time.Second,
			}),
		),
		grpcAddress: config.GRPCAddress,
	}

	reflection.Register(g.grpcServer)
	pb.RegisterRecorderServer(g.grpcServer, g)
	otelcollector.RegisterTraceServiceServer(g.grpcServer, g)

	return g, nil
}

func (g *Recorder) Run() error {
	lis, err := net.Listen("tcp", g.grpcAddress)
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	if err := g.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc serve: %w", err)
	}

	return nil
}

func (g *Recorder) Shutdown() error {
	g.grpcServer.GracefulStop()

	return nil
}

func (g *Recorder) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	id, err := uuid.FromBytes(r.Id)
	if err != nil {
		return nil, fmt.Errorf("id from bytes: %w", err)
	}

	event, err := g.recorder.Search(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &pb.SearchResponse{
		Event: event,
	}, nil
}

func (g *Recorder) CreateEvents(ctx context.Context, req *pb.CreateEventsRequest) (*pb.CreateEventsResponse, error) {
	for _, event := range req.Events {
		if _, err := uuid.FromBytes(event.Id); err != nil {
			return nil, fmt.Errorf("id from bytes: %w", err)
		}
	}

	if err := g.recorder.Write(ctx, req.Events); err != nil {
		return nil, fmt.Errorf("write events: %w", err)
	}

	return &pb.CreateEventsResponse{}, nil
}

func (g *Recorder) Export(ctx context.Context, req *otelcollector.ExportTraceServiceRequest) (*otelcollector.ExportTraceServiceResponse, error) {
	response, err := g.otelProxy.Grep(ctx, req.ResourceSpans)
	if err != nil {
		return nil, fmt.Errorf("grep: %w", err)
	}

	return response, nil
}
