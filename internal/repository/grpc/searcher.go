package grpc

import (
	"context"
	"fmt"

	"github.com/tekig/clerk/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Searcher struct {
	conn   *grpc.ClientConn
	client pb.SearcherClient
}

func NewSearcher(address string) (*Searcher, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}

	return &Searcher{
		conn:   conn,
		client: pb.NewSearcherClient(conn),
	}, nil
}

func (s *Searcher) AppendBlock(ctx context.Context, name string) error {
	if _, err := s.client.AppendBlock(ctx, &pb.AppendBlockRequest{
		Name: name,
	}); err != nil {
		return fmt.Errorf("append block: %w", err)
	}

	return nil
}

func (s *Searcher) Close() error {
	return s.conn.Close()
}
