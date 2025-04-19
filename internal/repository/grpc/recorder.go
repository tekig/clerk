package grpc

import (
	"context"
	"fmt"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Recorder struct {
	conn   *grpc.ClientConn
	client pb.RecorderClient
}

func NewRecorder(address string) (*Recorder, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}

	return &Recorder{
		conn:   conn,
		client: pb.NewRecorderClient(conn),
	}, nil
}

func (r *Recorder) Search(ctx context.Context, id uuid.UUID) (*pb.Event, error) {
	res, err := r.client.Search(ctx, &pb.SearchRequest{
		Id: id[:],
	})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return res.Event, nil
}

func (r *Recorder) Close() error {
	return r.conn.Close()
}
