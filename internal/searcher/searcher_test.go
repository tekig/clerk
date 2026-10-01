package searcher

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tekig/clerk/internal/block2"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/repository"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/repository/mock"
	"github.com/tekig/clerk/internal/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func newEvent() *pb.Event {
	id := uuid.New()

	return &pb.Event{
		Id: id[:],
		Attributes: []*pb.Attribute{{
			Key:   "key",
			Value: &pb.Attribute_AsString{AsString: "value"},
		}},
	}
}

// writeBlock writes a closed block with n events, as the recorder exports it.
func writeBlock(t *testing.T, s repository.Storage, name string, n int) []*pb.Event {
	t.Helper()

	b, err := block2.NewBlock(s, name, block2.MaxChunkSize(512))
	if err != nil {
		t.Fatalf("new block: %v", err)
	}

	events := make([]*pb.Event, 0, n)
	for range n {
		event := newEvent()
		if err := b.Write(event); err != nil {
			t.Fatalf("write: %v", err)
		}
		events = append(events, event)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("close block: %v", err)
	}

	return events
}

func newSearcher(t *testing.T, s repository.Storage, cache repository.Cache) *Searcher {
	t.Helper()

	searcher, err := NewSearcher(s, cache)
	if err != nil {
		t.Fatalf("new searcher: %v", err)
	}
	t.Cleanup(func() {
		searcher.Close()
	})

	return searcher
}

func TestSearcher_SearchBlocks(t *testing.T) {
	ctx := context.Background()
	storage := mem.NewStorage()
	searcher := newSearcher(t, storage, mem.NewCache())

	var events []*pb.Event
	for i := range 3 {
		name := fmt.Sprintf("block-%d", i)
		events = append(events, writeBlock(t, storage, name, 50)...)
		if err := searcher.AppendBlock(ctx, name); err != nil {
			t.Fatalf("append block: %v", err)
		}
	}

	for i, event := range events {
		got, err := searcher.Search(ctx, uuid.UUID(event.Id))
		if err != nil {
			t.Fatalf("search #%d: %v", i, err)
		}
		if !proto.Equal(got, event) {
			t.Fatalf("search #%d = %v, want %v", i, got, event)
		}
	}

	if _, err := searcher.Search(ctx, uuid.New()); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("search unknown id: err = %v, want %v", err, entity.ErrNotFound)
	}
}

func TestSearcher_AppendBlockMissing(t *testing.T) {
	searcher := newSearcher(t, mem.NewStorage(), mem.NewCache())

	if err := searcher.AppendBlock(context.Background(), "missing"); err == nil {
		t.Errorf("append missing block: expected error")
	}
}

func TestSearcher_RestoreOnStart(t *testing.T) {
	ctx := context.Background()
	storage := mem.NewStorage()
	events := writeBlock(t, storage, "block", 10)

	// Blocks are restored in background after start
	searcher := newSearcher(t, storage, mem.NewCache())

	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := searcher.Search(ctx, uuid.UUID(events[0].Id))
		if err == nil {
			if !proto.Equal(got, events[0]) {
				t.Errorf("search = %v, want %v", got, events[0])
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("block is not restored: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSearcher_Cache(t *testing.T) {
	ctx := context.Background()
	storage := mem.NewStorage()
	cache := mem.NewCache()
	searcher := newSearcher(t, storage, cache)

	events := writeBlock(t, storage, "block", 10)
	if err := searcher.AppendBlock(ctx, "block"); err != nil {
		t.Fatalf("append block: %v", err)
	}

	// Found event is cached
	if _, err := searcher.Search(ctx, uuid.UUID(events[0].Id)); err != nil {
		t.Fatalf("search: %v", err)
	}
	if cache.Get(ctx, uuid.UUID(events[0].Id)) == nil {
		t.Errorf("found event is not cached")
	}

	// Cached event is returned without storage
	cached := newEvent()
	cache.Set(ctx, cached)
	got, err := searcher.Search(ctx, uuid.UUID(cached.Id))
	if err != nil {
		t.Fatalf("search cached: %v", err)
	}
	if got != cached {
		t.Errorf("search cached = %v, want %v", got, cached)
	}
}

func TestSearcher_Recorders(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	storage := mem.NewStorage()
	searcher := newSearcher(t, storage, mem.NewCache())

	stored := writeBlock(t, storage, "block", 10)[0]
	if err := searcher.AppendBlock(ctx, "block"); err != nil {
		t.Fatalf("append block: %v", err)
	}

	recent := newEvent()

	failed := mock.NewMockRecorder(ctrl)
	failed.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, errors.New("unavailable")).AnyTimes()
	failed.EXPECT().Close().Return(nil).AnyTimes()

	active := mock.NewMockRecorder(ctrl)
	active.EXPECT().Search(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*pb.Event, error) {
		if id == uuid.UUID(recent.Id) {
			return recent, nil
		}
		return nil, entity.ErrNotFound
	}).AnyTimes()
	active.EXPECT().Close().Return(nil).AnyTimes()

	searcher.recorders = func() ([]repository.Recorder, error) {
		return []repository.Recorder{failed, active}, nil
	}

	t.Run("event from recorder", func(t *testing.T) {
		got, err := searcher.Search(ctx, uuid.UUID(recent.Id))
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if got != recent {
			t.Errorf("search = %v, want %v", got, recent)
		}
	})

	t.Run("not in recorders, found in storage", func(t *testing.T) {
		got, err := searcher.Search(ctx, uuid.UUID(stored.Id))
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if !proto.Equal(got, stored) {
			t.Errorf("search = %v, want %v", got, stored)
		}
	})

	t.Run("recorders unavailable", func(t *testing.T) {
		searcher.recorders = func() ([]repository.Recorder, error) {
			return nil, errors.New("lookup failed")
		}

		if _, err := searcher.Search(ctx, uuid.New()); !errors.Is(err, entity.ErrNotFound) {
			t.Errorf("search: err = %v, want %v", err, entity.ErrNotFound)
		}
	})
}
