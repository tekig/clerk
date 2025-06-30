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
	Read(ctx context.Context, name string) (io.ReadCloser, error)
	ReadRange(ctx context.Context, name string, offset, size int) (io.ReadCloser, error)
	SaveBlock(ctx context.Context, dir string) error
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
