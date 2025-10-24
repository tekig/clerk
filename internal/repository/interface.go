package repository

import (
	"context"
	"io"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
)

//go:generate mockgen -source=interface.go -package=mock -destination=mock/mock.go

type Storage interface {
	Blocks(ctx context.Context) ([]string, error)
	Read(ctx context.Context, block, name string) (io.ReadCloser, error)
	// size -1 reads the file completely
	ReadRange(ctx context.Context, block, name string, offset, size int) (io.ReadCloser, error)
	Write(ctx context.Context, block, name string) (io.WriteCloser, error)
}

type Recorder interface {
	Search(ctx context.Context, id uuid.UUID) (*pb.Event, error)
	Close() error
}

type Searcher interface {
	AppendBlock(ctx context.Context, name string) error
	Close() error
}

type Cache interface {
	Get(ctx context.Context, id uuid.UUID) *pb.Event
	Set(ctx context.Context, event *pb.Event)
}
