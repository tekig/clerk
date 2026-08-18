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
	"github.com/tekig/clerk/internal/x/xtime"
)

type Recorder struct {
	mu sync.Mutex

	block    *block2.Block
	searcher repository.Searcher
	exportes sync.WaitGroup

	blocksDir    string
	maxBlockSize int
	maxChunkSize *int
	// maxBlockAge defines the maximum allowed time to load a block.
	// If exceeded, the block will be forcibly recreated.
	maxBlockAge *xtime.Ticker

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

// Sets the maximum block assembly time, after which the block will be forced to recreate
func MaxBlockAge(d time.Duration) Option {
	return func(r *Recorder) {
		r.maxBlockAge = xtime.NewTicker(d)
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

	go r.maxAge()

	return r, nil
}

func (r *Recorder) Write(ctx context.Context, events []*pb.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.block == nil {
		b, err := r.newBlock()
		if err != nil {
			return fmt.Errorf("new block: %w", err)
		}

		r.block = b
	}

	for _, event := range events {
		if r.block.WritedSize() >= r.maxBlockSize {
			r.exportes.Add(1)
			go func(b *block2.Block) {
				defer r.exportes.Done()

				ctx, l := logger.NewLogger(context.Background())

				n := time.Now()
				err := r.export(ctx, b)

				attrs := []slog.Attr{
					slog.String("duration", time.Since(n).String()),
					slog.String("cause", "max block size"),
				}

				level := slog.LevelInfo
				if err != nil {
					level = slog.LevelError
					attrs = append(attrs, slog.String("error", err.Error()))
				}

				l.Log(level, "export block", attrs...)
			}(r.block)

			b, err := r.newBlock()
			if err != nil {
				return fmt.Errorf("recreate block: %w", err)
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
	if r.block == nil {
		r.mu.Unlock()
		return nil, fmt.Errorf("block is empty")
	}
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

	if r.maxBlockAge != nil {
		r.maxBlockAge.Close()
	}

	if r.block != nil && r.block.WritedSize() != 0 {
		if err := r.export(context.Background(), r.block); err != nil {
			return fmt.Errorf("export block: %w", err)
		}
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
	defer func() {
		logger.WithAttrs(ctx, slog.Any("export", slog.GroupValue(attrs...)))
	}()

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

	return nil
}

func (r *Recorder) maxAge() {
	if r.maxBlockAge == nil {
		return
	}

	r.exportes.Add(1)
	defer r.exportes.Done()

	for range r.maxBlockAge.C() {
		r.mu.Lock()

		b := r.block
		r.block = nil

		r.mu.Unlock()

		if b == nil {
			continue
		}

		ctx, l := logger.NewLogger(context.Background())

		n := time.Now()
		err := r.export(ctx, b)

		attrs := []slog.Attr{
			slog.String("duration", time.Since(n).String()),
			slog.String("cause", "max recording time"),
		}

		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		l.Log(level, "export block", attrs...)
	}
}
