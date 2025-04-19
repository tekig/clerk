package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/tekig/clerk/internal/logger"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/recorder"
	"github.com/tekig/clerk/internal/uuid"
	otelcollector "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var _ pb.RecorderServer = (*Recorder)(nil)

type Recorder struct {
	recorder   *recorder.Recorder
	server     *grpc.Server
	address    string
	targetOTEL otelcollector.TraceServiceClient
	pb.UnimplementedRecorderServer
	otelcollector.UnimplementedTraceServiceServer
}

type RecorderConfig struct {
	Recorder *recorder.Recorder
	Address  string
}

func NewRecorder(config RecorderConfig) (*Recorder, error) {
	g := &Recorder{
		recorder: config.Recorder,
		server: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				logger.UnaryServerInterceptor(),
			),
		),
		address: config.Address,
	}

	reflection.Register(g.server)
	pb.RegisterRecorderServer(g.server, g)

	return g, nil
}

func (g *Recorder) Run() error {
	lis, err := net.Listen("tcp", g.address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if err := g.server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (g *Recorder) Shutdown() error {
	g.server.GracefulStop()

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
	var events = make([]*pb.Event, 0, len(req.ResourceSpans))
	for _, span := range req.ResourceSpans {
		id := uuid.New()
		event := &pb.Event{
			Id:         id[:],
			Attributes: make([]*pb.Attribute, 0, len(span.Resource.Attributes)),
		}
		for _, attribute := range span.Resource.Attributes {
			var eventAttribute *pb.Attribute
			switch v := attribute.Value.Value.(type) {
			case *v1.AnyValue_StringValue:
				eventAttribute = &pb.Attribute{
					Key: attribute.Key,
					Value: &pb.Attribute_AsString{
						AsString: v.StringValue,
					},
				}
			case *v1.AnyValue_BoolValue:
				eventAttribute = &pb.Attribute{
					Key: attribute.Key,
					Value: &pb.Attribute_AsBool{
						AsBool: v.BoolValue,
					},
				}
			case *v1.AnyValue_IntValue:
				eventAttribute = &pb.Attribute{
					Key: attribute.Key,
					Value: &pb.Attribute_AsInt64{
						AsInt64: v.IntValue,
					},
				}
			case *v1.AnyValue_DoubleValue:
				eventAttribute = &pb.Attribute{
					Key: attribute.Key,
					Value: &pb.Attribute_AsDouble{
						AsDouble: v.DoubleValue,
					},
				}
			default:
				eventAttribute = &pb.Attribute{
					Key: attribute.Key,
					Value: &pb.Attribute_AsString{
						AsString: fmt.Sprintf("unsupport attribute %T", v),
					},
				}
			}
			span.Resource.Attributes = []*v1.KeyValue{
				{
					Key: "event_id",
					Value: &v1.AnyValue{
						Value: &v1.AnyValue_StringValue{
							StringValue: id.String(),
						},
					},
				},
			}
			event.Attributes = append(event.Attributes, eventAttribute)
		}
		events = append(events, event)
	}

	if err := g.recorder.Write(ctx, events); err != nil {
		return nil, fmt.Errorf("record events: %w", err)
	}

	res, err := g.targetOTEL.Export(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("target otel export: %w", err)
	}

	return res, nil
}
