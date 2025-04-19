package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/tekig/clerk/internal/logger"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/searcher"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Searcher struct {
	searcher *searcher.Searcher
	server   *grpc.Server
	address  string
	pb.UnimplementedSearcherServer
}

type SearcherConfig struct {
	Searcher *searcher.Searcher
	Address  string
}

func NewSearcher(config SearcherConfig) (*Searcher, error) {
	g := &Searcher{
		searcher: config.Searcher,
		server: grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				logger.UnaryServerInterceptor(),
			),
		),
		address: config.Address,
	}

	reflection.Register(g.server)
	pb.RegisterSearcherServer(g.server, g)

	return g, nil
}

func (g *Searcher) Run() error {
	lis, err := net.Listen("tcp", g.address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if err := g.server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (g *Searcher) Shutdown() error {
	g.server.GracefulStop()

	return nil
}

func (g *Searcher) Search(ctx context.Context, r *pb.SearchRequest) (*pb.SearchResponse, error) {
	id, err := uuid.FromBytes(r.Id)
	if err != nil {
		return nil, fmt.Errorf("id from bytes: %w", err)
	}

	event, err := g.searcher.Search(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &pb.SearchResponse{
		Event: event,
	}, nil
}

func (g *Searcher) AppendBlock(ctx context.Context, r *pb.AppendBlockRequest) (*pb.AppendBlockResponse, error) {
	if err := g.searcher.AppendBlock(ctx, r.Name); err != nil {
		return nil, fmt.Errorf("append block: %w", err)
	}

	return &pb.AppendBlockResponse{}, nil
}
