package block2

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bits-and-blooms/bloom"
	"github.com/golang/snappy"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/repository/mem"
	"github.com/tekig/clerk/internal/uuid"
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

func writeEvents(t *testing.T, b *Block, n int) []*pb.Event {
	t.Helper()

	events := make([]*pb.Event, 0, n)
	for range n {
		event := newEvent()
		if err := b.Write(event); err != nil {
			t.Fatalf("write: %v", err)
		}
		events = append(events, event)
	}

	return events
}

func readChunks(t *testing.T, s *mem.Storage, block string) []*pb.Index_Chunk {
	t.Helper()

	r, err := s.Read(context.Background(), block, entity.NameIndex)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	defer r.Close()

	var chunks []*pb.Index_Chunk
	for {
		chunk := &pb.Index_Chunk{}
		if err := Decode(chunk, r); err != nil {
			if errors.Is(err, io.EOF) {
				return chunks
			}
			t.Fatalf("decode index: %v", err)
		}
		chunks = append(chunks, chunk)
	}
}

func TestBlock_Close(t *testing.T) {
	ctx := context.Background()
	s := mem.NewStorage()

	b, err := NewBlock(s, "block", MaxChunkSize(1024))
	if err != nil {
		t.Fatalf("new block: %v", err)
	}

	events := writeEvents(t, b, 200)
	if b.WritedSize() == 0 {
		t.Errorf("written size is zero")
	}

	if err := b.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if b.CompressedSize() == 0 {
		t.Errorf("compressed size is zero")
	}

	t.Run("index and data", func(t *testing.T) {
		chunks := readChunks(t, s, "block")
		if len(chunks) < 2 {
			t.Fatalf("chunks = %d, want several with small chunk size", len(chunks))
		}

		// Every chunk mark points to the snappy stream with the chunk events, in write order
		var i int
		for _, chunk := range chunks {
			r, err := s.ReadRange(ctx, "block", entity.NameData, int(chunk.Mark.Offset), int(chunk.Mark.Size))
			if err != nil {
				t.Fatalf("read chunk: %v", err)
			}

			snap := snappy.NewReader(r)
			for _, id := range chunk.Ids {
				event := &pb.Event{}
				if err := Decode(event, snap); err != nil {
					t.Fatalf("decode event #%d: %v", i, err)
				}
				if !proto.Equal(event, events[i]) {
					t.Fatalf("event #%d = %v, want %v", i, event, events[i])
				}
				if uuid.UUID(id) != uuid.UUID(events[i].Id) {
					t.Fatalf("index id #%d does not match event", i)
				}
				i++
			}
			r.Close()
		}
		if i != len(events) {
			t.Errorf("indexed events = %d, want %d", i, len(events))
		}
	})

	t.Run("filters", func(t *testing.T) {
		r, err := s.Read(ctx, "block", entity.NameBloom)
		if err != nil {
			t.Fatalf("read bloom: %v", err)
		}
		defer r.Close()

		filters := &pb.Filters{}
		if err := Decode(filters, r); err != nil {
			t.Fatalf("decode filters: %v", err)
		}

		bl := &bloom.BloomFilter{}
		if _, err := bl.ReadFrom(bytes.NewReader(filters.Bloom)); err != nil {
			t.Fatalf("decode bloom: %v", err)
		}

		start := filters.GetTimeMillis().GetStart()
		end := filters.GetTimeMillis().GetEnd()
		for _, event := range events {
			if !bl.Test(event.Id) {
				t.Errorf("bloom does not contain written id")
			}

			ms := uuid.UUID(event.Id).Time().UnixMilli()
			if ms < start || ms > end {
				t.Errorf("id time %d is out of block range [%d, %d]", ms, start, end)
			}
		}
	})

	t.Run("closed", func(t *testing.T) {
		if err := b.Write(newEvent()); err == nil {
			t.Errorf("write to closed block: expected error")
		}
		if _, err := b.Search(ctx, uuid.UUID(events[0].Id)); err == nil {
			t.Errorf("search in closed block: expected error")
		}
		if err := b.Close(); err == nil {
			t.Errorf("second close: expected error")
		}
	})
}

func TestBlock_Search(t *testing.T) {
	ctx := context.Background()

	b, err := NewBlock(mem.NewStorage(), "block", MaxChunkSize(1024))
	if err != nil {
		t.Fatalf("new block: %v", err)
	}
	defer b.Close()

	// The last chunk is not flushed yet, search only in closed chunks
	events := writeEvents(t, b, 200)
	for i, event := range events[:100] {
		got, err := b.Search(ctx, uuid.UUID(event.Id))
		if err != nil {
			t.Fatalf("search #%d: %v", i, err)
		}
		if !proto.Equal(got, event) {
			t.Errorf("search #%d = %v, want %v", i, got, event)
		}
	}

	if _, err := b.Search(ctx, uuid.New()); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("search unknown id: err = %v, want %v", err, entity.ErrNotFound)
	}
}

func TestBlock_ConcurrentWriteSearch(t *testing.T) {
	b, err := NewBlock(mem.NewStorage(), "block", MaxChunkSize(1024))
	if err != nil {
		t.Fatalf("new block: %v", err)
	}
	defer b.Close()

	// The first chunk is closed and must be found at any moment
	target := writeEvents(t, b, 100)[0]

	// Ids of the current chunk are also searched while chunks are switched
	var last atomic.Pointer[[]byte]
	last.Store(&target.Id)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()

		for range 2000 {
			event := newEvent()
			if err := b.Write(event); err != nil {
				t.Errorf("write: %v", err)
				return
			}
			last.Store(&event.Id)
		}
	}()
	go func() {
		defer wg.Done()

		for range 1000 {
			if _, err := b.Search(context.Background(), uuid.UUID(target.Id)); err != nil {
				t.Errorf("search closed chunk: %v", err)
				return
			}

			// Data of the current chunk may still be in the snappy buffer,
			// only concurrent access is checked
			_, _ = b.Search(context.Background(), uuid.UUID(*last.Load()))
		}
	}()
	wg.Wait()
}
