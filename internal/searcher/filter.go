package searcher

import (
	"time"

	"github.com/bits-and-blooms/bloom"
)

type filters struct {
	Bloom func() (*bloom.BloomFilter, error)
	Time  timeRange
}

type timeRange struct {
	Start time.Time
	End   time.Time
}
