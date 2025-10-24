package recorder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/tekig/clerk/internal/block2"
	"github.com/tekig/clerk/internal/logger"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/repository"
	"github.com/tekig/clerk/internal/uuid"
)

type Recorder struct {
	mu sync.Mutex

	block    *block2.Block
	searcher repository.Searcher
	exportes sync.WaitGroup

	blocksDir    string
	maxBlockSize int
	maxChunkSize *int

	storage repository.Storage
}

type Option func(r *Recorder)

func MaxBlockSize(s int) Option {
	return func(r *Recorder) {
		r.maxBlockSize = s
	}
}

func BlocksDir(d string) Option {
	return func(r *Recorder) {
		r.blocksDir = d
	}
}

func MaxChunkSize(s int) Option {
	return func(r *Recorder) {
		r.maxChunkSize = &s
	}
}

func NewRecorder(storage repository.Storage, searcher repository.Searcher, options ...Option) (*Recorder, error) {
	tmp := os.TempDir()

	r := &Recorder{
		blocksDir:    tmp,
		maxBlockSize: 1 * 1024 * 1024 * 1024,
		storage:      storage,
		searcher:     searcher,
	}

	for _, o := range options {
		o(r)
	}

	b, err := r.newBlock()
	if err != nil {
		return nil, fmt.Errorf("new block: %w", err)
	}

	r.block = b

	return r, nil
}

func (r *Recorder) Write(ctx context.Context, events []*pb.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, event := range events {
		if r.block.WritedSize() >= r.maxBlockSize {
			r.exportes.Add(1)
			go func(b *block2.Block) {
				defer r.exportes.Done()

				ctx, l := logger.NewLogger(context.Background())

				n := time.Now()
				err := r.export(ctx, b)

				var attrs = []slog.Attr{
					slog.String("duration", time.Since(n).String()),
				}

				var level = slog.LevelInfo
				if err != nil {
					level = slog.LevelError
					attrs = append(attrs, slog.String("error", err.Error()))
				}

				l.Log(level, "export block", attrs...)
			}(r.block)

			b, err := r.newBlock()
			if err != nil {
				return fmt.Errorf("new block: %w", err)
			}

			r.block = b
		}

		if err := r.block.Write(event); err != nil {
			return fmt.Errorf("write block: %w", err)
		}
	}

	return nil
}

func (r *Recorder) Search(ctx context.Context, id uuid.UUID) (*pb.Event, error) {
	r.mu.Lock()
	b := r.block
	r.mu.Unlock()

	event, err := b.Search(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return event, nil
}

func (r *Recorder) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.block.WritedSize() == 0 {
		return nil
	}

	if err := r.export(context.Background(), r.block); err != nil {
		return fmt.Errorf("export block: %w", err)
	}

	r.exportes.Wait()

	return nil
}

func (r *Recorder) newBlock() (*block2.Block, error) {
	var options []block2.BlockOption
	if r.maxChunkSize != nil {
		options = append(options, block2.MaxChunkSize(*r.maxChunkSize))
	}

	b, err := block2.NewBlock(r.storage, uuid.New().String(), options...)
	if err != nil {
		return nil, fmt.Errorf("new block: %w", err)
	}

	return b, nil
}

func (r *Recorder) export(ctx context.Context, block *block2.Block) error {
	var attrs []slog.Attr

	t1 := time.Now()
	if err := block.Close(); err != nil {
		return fmt.Errorf("close block: %w", err)
	}
	attrs = append(attrs,
		slog.String("block_close", time.Since(t1).String()),
		slog.Int("block_origin_size", block.WritedSize()),
		slog.Int("block_compressed_size", block.CompressedSize()),
		slog.Float64("block_compressed_rate", float64(block.WritedSize())/float64(block.CompressedSize())),
	)

	t2 := time.Now()
	if err := r.searcher.AppendBlock(ctx, block.ID()); err != nil {
		return fmt.Errorf("append block: %w", err)
	}
	attrs = append(attrs, slog.String("search_notify", time.Since(t2).String()))

	logger.WithAttrs(ctx, slog.Any("export", slog.GroupValue(attrs...)))

	return nil
}
