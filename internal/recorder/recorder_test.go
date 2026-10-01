package recorder

import (
	"context"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/tekig/clerk/internal/block2"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/repository/mock"
	"github.com/tekig/clerk/internal/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func newEvents(n int) []*pb.Event {
	events := make([]*pb.Event, 0, n)
	for range n {
		id := uuid.New()
		events = append(events, &pb.Event{
			Id: id[:],
			Attributes: []*pb.Attribute{{
				Key:   "key",
				Value: &pb.Attribute_AsString{AsString: "value"},
			}},
		})
	}

	return events
}

// appendedBlocks collects blocks the searcher is notified about.
type appendedBlocks struct {
	mu     sync.Mutex
	blocks []string
}

func (a *appendedBlocks) expect(s *mock.MockSearcher) {
	s.EXPECT().AppendBlock(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, name string) error {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.blocks = append(a.blocks, name)

		return nil
	}).AnyTimes()
}

func (a *appendedBlocks) get() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	return slices.Clone(a.blocks)
}

// indexedIds reads all ids from the block index.
func indexedIds(t *testing.T, s *mem.Storage, block string) [][]byte {
	t.Helper()

	r, err := s.Read(context.Background(), block, entity.NameIndex)
	if err != nil {
		t.Fatalf("read index `%s`: %v", block, err)
	}
	defer r.Close()

	var ids [][]byte
	for {
		chunk := &pb.Index_Chunk{}
		if err := block2.Decode(chunk, r); err != nil {
			if errors.Is(err, io.EOF) {
				return ids
			}
			t.Fatalf("decode index: %v", err)
		}
		ids = append(ids, chunk.Ids...)
	}
}

func TestRecorder_WriteRotateShutdown(t *testing.T) {
	ctx := context.Background()
	storage := mem.NewStorage()
	searcher := mock.NewMockSearcher(gomock.NewController(t))
	var appended appendedBlocks
	appended.expect(searcher)

	r, err := NewRecorder(storage, searcher, MaxBlockSize(1024), MaxChunkSize(256))
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}

	events := newEvents(100)
	for i := 0; i < len(events); i += 10 {
		if err := r.Write(ctx, events[i:i+10]); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	if err := r.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// Blocks are rotated by size, and every block, including the last one, is exported
	blocks := appended.get()
	if len(blocks) < 2 {
		t.Fatalf("exported blocks = %d, want several with small block size", len(blocks))
	}

	stored, err := storage.Blocks(ctx)
	if err != nil {
		t.Fatalf("blocks: %v", err)
	}
	slices.Sort(blocks)
	if !slices.Equal(blocks, stored) {
		t.Errorf("notified blocks %v, stored blocks %v", blocks, stored)
	}

	var ids [][]byte
	for _, block := range blocks {
		if _, err := storage.Read(ctx, block, entity.NameBloom); err != nil {
			t.Errorf("block `%s` has no bloom: %v", block, err)
		}
		ids = append(ids, indexedIds(t, storage, block)...)
	}
	if len(ids) != len(events) {
		t.Fatalf("indexed events = %d, want %d", len(ids), len(events))
	}
	for i, event := range events {
		if uuid.UUID(ids[i]) != uuid.UUID(event.Id) {
			t.Fatalf("event #%d is indexed out of order", i)
		}
	}
}

func TestRecorder_Search(t *testing.T) {
	ctx := context.Background()
	searcher := mock.NewMockSearcher(gomock.NewController(t))
	searcher.EXPECT().AppendBlock(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	r, err := NewRecorder(mem.NewStorage(), searcher, MaxChunkSize(256))
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	defer r.Shutdown()

	if _, err := r.Search(ctx, uuid.New()); err == nil {
		t.Errorf("search without block: expected error")
	}

	events := newEvents(100)
	if err := r.Write(ctx, events); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The last chunk may still be in the snappy buffer, the first event is in a closed chunk
	got, err := r.Search(ctx, uuid.UUID(events[0].Id))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !proto.Equal(got, events[0]) {
		t.Errorf("search = %v, want %v", got, events[0])
	}

	if _, err := r.Search(ctx, uuid.New()); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("search unknown id: err = %v, want %v", err, entity.ErrNotFound)
	}
}

func TestRecorder_MaxBlockAge(t *testing.T) {
	ctx := context.Background()
	storage := mem.NewStorage()
	searcher := mock.NewMockSearcher(gomock.NewController(t))
	var appended appendedBlocks
	appended.expect(searcher)

	r, err := NewRecorder(storage, searcher, MaxBlockAge(50*time.Millisecond))
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}

	if err := r.Write(ctx, newEvents(10)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// The block is exported by age without reaching the size limit
	deadline := time.Now().Add(5 * time.Second)
	for len(appended.get()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(appended.get()) != 1 {
		t.Fatalf("exported blocks = %d, want 1", len(appended.get()))
	}

	// Next write starts a new block
	if err := r.Write(ctx, newEvents(10)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := r.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	blocks := appended.get()
	if len(blocks) != 2 || blocks[0] == blocks[1] {
		t.Errorf("exported blocks = %v, want 2 different", blocks)
	}
}

func TestRecorder_ShutdownEmpty(t *testing.T) {
	// Searcher is not notified when nothing was written
	searcher := mock.NewMockSearcher(gomock.NewController(t))

	r, err := NewRecorder(mem.NewStorage(), searcher)
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}

	if err := r.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
