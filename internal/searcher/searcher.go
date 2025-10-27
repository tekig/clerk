package searcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom"
	"github.com/golang/snappy"
	"github.com/tekig/clerk/internal/block2"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/logger"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/repository"
	"github.com/tekig/clerk/internal/repository/grpc"
	"github.com/tekig/clerk/internal/uuid"
)

type Searcher struct {
	recorders func() ([]repository.Recorder, error)
	filters   map[string]filters
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
		filters: make(map[string]filters),
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
		attr = append(attr, slog.String("duration", time.Since(t1).String()))

		l.Log(level, "restore index", attr...)
	}

	return s, nil
}

func (s *Searcher) Close() error {
	return os.RemoveAll(s.tmp)
}

func (s *Searcher) AppendBlock(ctx context.Context, name string) error {
	r, err := s.storage.Read(ctx, name, entity.NameBloom)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read filters: %w", err)
	}

	var pbFilters = &pb.Filters{}
	if err := block2.Decode(pbFilters, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("decode filters: %w", err)
	}

	fileName := path.Join(s.tmp, name)

	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("copy to local storage: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.filters[name] = filters{
		Bloom: func() (*bloom.BloomFilter, error) {
			f, err := os.Open(fileName)
			if err != nil {
				return nil, fmt.Errorf("open: %w", err)
			}
			defer f.Close()

			var data = &pb.Filters{}
			if err := block2.Decode(data, f); err != nil {
				return nil, fmt.Errorf("decode message: %w", err)
			}

			bl := &bloom.BloomFilter{}
			if _, err := bl.ReadFrom(bytes.NewBuffer(data.Bloom)); err != nil {
				return nil, fmt.Errorf("decode bloom: %w", err)
			}

			return bl, nil
		},
		Time: timeRange{
			Start: time.UnixMilli(pbFilters.GetTimeMillis().GetStart()),
			End:   time.UnixMilli(pbFilters.GetTimeMillis().GetEnd()),
		},
	}

	return nil
}

func (s *Searcher) Search(ctx context.Context, id uuid.UUID) (*pb.Event, error) {
	var attrs []slog.Attr
	defer func() {
		logger.WithAttrs(ctx, attrs...)
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

	if event := s.searchRecorder(ctx, id); event != nil {
		return event, nil
	}

	block, mark := s.searchIndexes(ctx, id)
	if block == nil {
		return nil, entity.ErrNotFound
	}

	t1 := time.Now()
	event, err := s.searchChunk(ctx, *block, mark, id)
	if err != nil {
		return nil, fmt.Errorf("search chunk: %w", err)
	}

	attrs = append(attrs, slog.String("chuck_duration", time.Since(t1).String()))

	return event, nil
}

func (s *Searcher) searchFilters(ctx context.Context, target uuid.UUID) []string {
	var attrs []slog.Attr
	defer func() {
		logger.WithAttrs(ctx, slog.Any("blooms", slog.GroupValue(attrs...)))
	}()

	var (
		candidats      []string
		candidatsTime  int
		candidatsBloom int
		bloomLoadDur   time.Duration
		t1             = time.Now()
	)
	for blockName, filters := range s.filters {
		// support old uuids that do not have time component
		if target.Time() != filters.Time.Start {
			if filters.Time.Start.After(target.Time()) || filters.Time.End.Before(target.Time()) {
				continue
			}
		}
		candidatsTime++

		t2 := time.Now()
		bl, err := filters.Bloom()
		if err != nil {
			attrs = append(attrs, slog.Group(
				blockName,
				slog.String("error", fmt.Errorf("load bloom: %w", err).Error()),
			))
			continue
		}
		bloomLoadDur += time.Since(t2)

		if bl.Test(target[:]) {
			candidats = append(candidats, blockName)
			candidatsBloom++
		}
	}
	attrs = append(
		attrs,
		slog.Group(
			"filters",
			slog.Int("candidats_total", len(s.filters)),
			slog.String("duration_total", time.Since(t1).String()),
			slog.Group(
				"time",
				slog.Int("candidats", candidatsTime),
			),
			slog.Group(
				"bloom",
				slog.String("duration_load", bloomLoadDur.String()),
				slog.Int("candidats", candidatsBloom),
			),
		),
	)

	return candidats
}

func (s *Searcher) searchRecorder(ctx context.Context, target uuid.UUID) *pb.Event {
	var attrs []slog.Attr
	defer func() {
		logger.WithAttrs(ctx, slog.Any("recorders", slog.GroupValue(attrs...)))
	}()

	recorders, err := s.recorders()
	if err != nil {
		attrs = append(attrs, slog.String("error", fmt.Errorf("recorders: %w", err).Error()))
	}
	defer func() {
		for _, r := range recorders {
			r.Close()
		}
	}()

	for i, r := range recorders {
		var attrGroup []any
		t1 := time.Now()

		event, err := r.Search(ctx, target)
		if err != nil {
			attrGroup = append(attrGroup, slog.String("error", fmt.Errorf("search: %w", err).Error()))
		}

		attrGroup = append(attrGroup, slog.String("duration", time.Since(t1).String()))

		attrs = append(attrs, slog.Group(
			fmt.Sprintf("%d", i),
			attrGroup...,
		))

		if event != nil {
			return event
		}
	}

	return nil
}

func (s *Searcher) searchIndexes(ctx context.Context, target uuid.UUID) (*string, *pb.Index_Chunk_Mark) {
	var attrs []slog.Attr
	defer func() {
		logger.WithAttrs(ctx, slog.Any("indexes", slog.GroupValue(attrs...)))
	}()

	var mark *pb.Index_Chunk_Mark
	var block *string
	for _, blockName := range s.searchFilters(ctx, target) {
		var attrGroup []any
		t1 := time.Now()

		markIndex, err := s.searchIndex(ctx, blockName, target)
		if err != nil {
			attrGroup = append(attrGroup, slog.String("error", fmt.Errorf("search: %w", err).Error()))
		}

		attrGroup = append(attrGroup,
			slog.String("duration", time.Since(t1).String()),
			slog.Bool("find", markIndex != nil),
		)

		if markIndex == nil {
			continue
		}

		attrs = append(attrs, slog.Group(
			blockName,
			attrGroup...,
		))

		block = &blockName
		mark = markIndex

		break
	}

	return block, mark
}

func (s *Searcher) searchIndex(ctx context.Context, blockName string, target uuid.UUID) (*pb.Index_Chunk_Mark, error) {
	r, err := s.storage.Read(ctx, blockName, entity.NameIndex)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	defer r.Close()

	for {
		var idx = &pb.Index_Chunk{}
		if err := block2.Decode(idx, r); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, entity.ErrNotFound
			}
			return nil, fmt.Errorf("decode: %w", err)
		}

		for _, id := range idx.Ids {
			if uuid.UUID(id) == target {
				return idx.Mark, nil
			}
		}
	}
}

func (s *Searcher) searchChunk(ctx context.Context, blockName string, mark *pb.Index_Chunk_Mark, target uuid.UUID) (*pb.Event, error) {
	f, err := s.storage.ReadRange(ctx, blockName, entity.NameData, int(mark.Offset), int(mark.Size))
	if err != nil {
		return nil, fmt.Errorf("read chunk `%s`: %w", blockName, err)
	}
	defer f.Close()

	r := snappy.NewReader(f)

	for {
		var m = &pb.Event{}
		if err := block2.Decode(m, r); err != nil {
			return nil, fmt.Errorf("decode: %w", err)
		}

		if uuid.UUID(m.Id) == target {
			return m, nil
		}
	}
}
