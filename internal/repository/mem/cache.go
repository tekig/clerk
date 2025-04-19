package mem

import (
	"context"
	"sync"
	"time"

	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
)

type Cache struct {
	m map[uuid.UUID]*cacheValue

	maxSize int
	mu      sync.Mutex
}

type cacheValue struct {
	lastVisitAt time.Time
	event       *pb.Event
}

type OptionCache func(c *Cache)

func MaxSizeCache(n int) OptionCache {
	return func(c *Cache) {
		c.maxSize = n
	}
}

func NewCache(options ...OptionCache) *Cache {
	c := &Cache{
		maxSize: 100,
		m:       make(map[uuid.UUID]*cacheValue, 0),
	}

	for _, option := range options {
		option(c)
	}

	return c
}

func (c *Cache) Get(ctx context.Context, id uuid.UUID) *pb.Event {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, ok := c.m[id]
	if !ok {
		return nil
	}

	value.lastVisitAt = time.Now()

	return value.event
}

func (c *Cache) Set(ctx context.Context, event *pb.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.m) > c.maxSize {
		var (
			oldestKey uuid.UUID
			oldestAt  = time.Now()
		)
		for k, v := range c.m {
			if v.lastVisitAt.After(oldestAt) {
				continue
			}
			oldestKey = k
			oldestAt = v.lastVisitAt
		}

		delete(c.m, oldestKey)
	}

	c.m[uuid.UUID(event.Id)] = &cacheValue{
		lastVisitAt: time.Now(),
		event:       event,
	}
}
