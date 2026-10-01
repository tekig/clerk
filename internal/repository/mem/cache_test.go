package mem

import (
	"context"
	"testing"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
)

func newCacheEvent() *pb.Event {
	id := uuid.New()

	return &pb.Event{Id: id[:]}
}

func TestCache_GetSet(t *testing.T) {
	ctx := context.Background()
	c := NewCache()

	event := newCacheEvent()
	if got := c.Get(ctx, uuid.UUID(event.Id)); got != nil {
		t.Errorf("get before set = %v, want nil", got)
	}

	c.Set(ctx, event)
	if got := c.Get(ctx, uuid.UUID(event.Id)); got != event {
		t.Errorf("get = %v, want %v", got, event)
	}
}

func TestCache_EvictOldest(t *testing.T) {
	ctx := context.Background()
	c := NewCache(MaxSizeCache(2))

	events := []*pb.Event{newCacheEvent(), newCacheEvent(), newCacheEvent()}
	for _, e := range events {
		c.Set(ctx, e)
	}

	// The first event is visited, so the second becomes the oldest
	c.Get(ctx, uuid.UUID(events[0].Id))

	c.Set(ctx, newCacheEvent())

	if c.Get(ctx, uuid.UUID(events[0].Id)) == nil {
		t.Errorf("recently visited event was evicted")
	}
	if c.Get(ctx, uuid.UUID(events[1].Id)) != nil {
		t.Errorf("oldest event was not evicted")
	}
	if len(c.m) != 3 {
		t.Errorf("cache size = %d, want 3", len(c.m))
	}
}
