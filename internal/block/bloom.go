package block

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/bits-and-blooms/bloom"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/protobuf/proto"
)

type Bloom struct {
	blooms    []*bloom.BloomFilter
	pointers  []IndexPointer
	chunkIDs  map[uuid.UUID]struct{}
	blockSize int

	mu sync.RWMutex
}

type IndexPointer struct {
	ChunkOffset int64
	ChunkSize   int64
}

func NewBloom() *Bloom {
	return &Bloom{
		chunkIDs: make(map[uuid.UUID]struct{}),
	}
}

func (b *Bloom) Read(r io.Reader) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	var filter = &pb.Bloom{}
	if err := proto.Unmarshal(data, filter); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	for _, chunk := range filter.Chunks {
		var g = &bloom.BloomFilter{}
		if _, err := g.ReadFrom(bytes.NewReader(chunk.Bloom)); err != nil {
			return fmt.Errorf("bloom read: %w", err)
		}

		b.blooms = append(b.blooms, g)
		b.pointers = append(b.pointers, IndexPointer{
			ChunkOffset: chunk.Offset,
			ChunkSize:   chunk.Size,
		})
	}

	return nil
}

func (b *Bloom) Write(w io.Writer) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var filter = &pb.Bloom{}
	for i := range len(b.blooms) {
		bl := b.blooms[i]
		po := b.pointers[i]

		var buf = bytes.NewBuffer(nil)
		if _, err := bl.WriteTo(buf); err != nil {
			return fmt.Errorf("write bloom: %w", err)
		}

		filter.Chunks = append(filter.Chunks, &pb.Bloom_Chunk{
			Bloom:  buf.Bytes(),
			Size:   po.ChunkSize,
			Offset: po.ChunkOffset,
		})
	}

	data, err := proto.Marshal(filter)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

func (b *Bloom) ChunkAppend(id uuid.UUID) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.chunkIDs[id] = struct{}{}
}

func (b *Bloom) ChunkClose(chunkSize int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	pointer := IndexPointer{
		ChunkOffset: int64(b.blockSize),
		ChunkSize:   int64(chunkSize),
	}

	bl := bloom.NewWithEstimates(uint(len(b.chunkIDs)), 0.01)
	for id := range b.chunkIDs {
		bl.Add(id[:])
	}

	b.blooms = append(b.blooms, bl)
	b.pointers = append(b.pointers, pointer)

	b.blockSize += chunkSize

	b.chunkIDs = make(map[uuid.UUID]struct{}, len(b.chunkIDs))
}

func (b *Bloom) Search(target uuid.UUID) []IndexPointer {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if _, ok := b.chunkIDs[target]; ok {
		return []IndexPointer{
			{
				ChunkOffset: int64(b.blockSize),
				ChunkSize:   -1,
			},
		}
	}

	var candidats = make([]IndexPointer, 0)
	for i, bl := range b.blooms {
		if !bl.Test(target[:]) {
			continue
		}

		candidats = append(candidats, b.pointers[i])
	}

	return candidats
}
