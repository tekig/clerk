package searcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/tekig/clerk/internal/block"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/logger"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/repository"
	"github.com/tekig/clerk/internal/repository/grpc"
	"github.com/tekig/clerk/internal/uuid"
)

type Searcher struct {
	recorders func() ([]repository.Recorder, error)
	indexs    map[string]func() (*block.Bloom, error)
	cache     repository.Cache
	tmp       string

	mu      sync.Mutex
	storage repository.Storage
}

type Option func(r *Searcher)

func Readers(address []string) Option {
	return func(r *Searcher) {

		r.recorders = func() ([]repository.Recorder, error) {
			var recorders []repository.Recorder
			for _, address := range address {
				host, port, err := net.SplitHostPort(address)
				if err != nil {
					return nil, fmt.Errorf("split `%s` host port: %w", address, err)
				}

				ips, err := net.LookupIP(host)
				if err != nil {
					return nil, fmt.Errorf("lookup `%s`: %w", address, err)
				}

				for _, ip := range ips {
					recoder, err := grpc.NewRecorder(net.JoinHostPort(ip.String(), port))
					if err != nil {
						return nil, fmt.Errorf("new recoder `%s`: %w", address, err)
					}

					recorders = append(recorders, recoder)
				}
			}

			return recorders, nil
		}
	}
}

func NewSearcher(storage repository.Storage, cache repository.Cache, options ...Option) (*Searcher, error) {
	tmp, err := os.MkdirTemp("", "index-*")
	if err != nil {
		return nil, fmt.Errorf("index mkdir temp: %w", err)
	}

	s := &Searcher{
		storage: storage,
		cache:   cache,
		indexs:  make(map[string]func() (*block.Bloom, error)),
		tmp:     tmp,
		recorders: func() ([]repository.Recorder, error) {
			return nil, nil
		},
	}

	for _, option := range options {
		option(s)
	}

	blocks, err := storage.Blocks(context.Background())
	if err != nil {
		return nil, fmt.Errorf("find blocks: %w", err)
	}

	for _, block := range blocks {
		var attr = []slog.Attr{
			slog.String("block", block),
		}

		ctx, l := logger.NewLogger(context.Background())
		var level = slog.LevelInfo
		t1 := time.Now()
		if err := s.AppendBlock(ctx, block); err != nil {
			level = slog.LevelWarn
			attr = append(attr, slog.String("error", err.Error()))
		}
		attr = append(attr, slog.Duration("duration", time.Since(t1)))

		l.Log(level, "restore index", attr...)
	}

	return s, nil
}

func (s *Searcher) Close() error {
	return os.RemoveAll(s.tmp)
}

func (s *Searcher) AppendBlock(ctx context.Context, name string) error {
	r, err := s.storage.Read(ctx, path.Join(name, entity.NameIndex))
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	defer r.Close()

	fileName := path.Join(s.tmp, name)

	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	s.mu.Lock()
	s.indexs[name] = func() (*block.Bloom, error) {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("open: %w", err)
		}
		defer f.Close()

		index := block.NewBloom()
		if err := index.Read(f); err != nil {
			return nil, fmt.Errorf("read index: %w", err)
		}

		return index, nil
	}
	s.mu.Unlock()

	return nil
}

func (s *Searcher) Search(ctx context.Context, id uuid.UUID) (*pb.Event, error) {
	var attrs []slog.Attr
	defer func() {
		logger.WithAttrs(ctx, slog.Any("search", slog.GroupValue(attrs...)))
	}()

	s.mu.Lock()
	defer s.mu.Unlock()

	event := s.cache.Get(ctx, id)
	if event != nil {
		attrs = append(attrs, slog.Bool("cache", true))

		return event, nil
	}

	event, err := s.search(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	s.cache.Set(ctx, event)
	attrs = append(attrs, slog.Bool("cache", false))

	return event, nil
}

func (s *Searcher) search(ctx context.Context, id uuid.UUID) (*pb.Event, error) {
	var attrs []slog.Attr
	defer func() {
		logger.WithAttrs(ctx, slog.Any("search", slog.GroupValue(attrs...)))
	}()

	recorders, err := s.recorders()
	if err != nil {
		return nil, fmt.Errorf("recorders: %w", err)
	}
	defer func() {
		for _, r := range recorders {
			r.Close()
		}
	}()

	for i, r := range recorders {
		var attrGroup []any
		t1 := time.Now()

		event, err := r.Search(ctx, id)
		if err != nil {
			attrGroup = append(attrGroup, slog.String("error", err.Error()))
		}

		attrGroup = append(attrGroup, slog.Duration("duration", time.Since(t1)))

		attrs = append(attrs, slog.Group(
			fmt.Sprintf("recorder_%d", i),
			attrGroup...,
		))

		if event != nil {
			return event, nil
		}
	}

	t2 := time.Now()
	var candidats = map[string][]block.IndexPointer{}
	var countCandidats int
	for name, indexFn := range s.indexs {
		index, err := indexFn()
		if err != nil {
			attrs = append(attrs, slog.Group(
				fmt.Sprintf("%s_index", name),
				slog.String("error", err.Error()),
			))
		}

		c := index.Search(id)
		if len(c) > 0 {
			countCandidats += len(c)
			candidats[name] = c
		}
	}
	attrs = append(
		attrs,
		slog.Group(
			"index",
			slog.Duration("duration", time.Since(t2)),
			slog.Int("candidats", countCandidats),
		),
	)

	for blockName, c := range candidats {
		for _, chunk := range c {
			var attrGroup []any
			t3 := time.Now()

			event, err := s.searchChunk(ctx, id, blockName, chunk)
			if err != nil {
				attrGroup = append(attrGroup, slog.String("error", err.Error()))
			}

			attrGroup = append(attrGroup, slog.Duration("duration", time.Since(t3)))

			attrs = append(attrs, slog.Group(
				fmt.Sprintf("%s_%d", blockName, chunk.ChunkOffset),
				attrGroup...,
			))

			if event != nil {
				return event, nil
			}
		}
	}

	return nil, entity.ErrNotFound
}

func (s *Searcher) searchChunk(ctx context.Context, id uuid.UUID, blockName string, chunk block.IndexPointer) (*pb.Event, error) {
	r, err := s.storage.ReadRange(ctx, path.Join(blockName, entity.NameData), int(chunk.ChunkOffset), int(chunk.ChunkSize))
	if err != nil {
		return nil, fmt.Errorf("read chunk `%s`: %w", blockName, err)
	}
	defer r.Close()

	event, err := block.SearchChunk(ctx, r, id)
	if err != nil {
		return nil, fmt.Errorf("search chunk: %w", err)
	}

	return event, nil
}
