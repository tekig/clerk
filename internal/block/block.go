package block

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/protobuf/proto"
)

type Block struct {
	path         string
	block        *os.File
	blockWriter  *WriteCounter
	index        *Bloom
	chunk        *Chunk
	size         int
	maxChunkSize int
	closed       bool
	buf          []byte
	wmu          sync.Mutex   // mutex from write
	rmu          sync.RWMutex // mutex from read
}

type Config struct {
	BlockPath string
	BlockName string
	ChunkSize int
}

type BlockOption func(*Block)

func MaxChunkSize(size int) BlockOption {
	return func(b *Block) {
		b.maxChunkSize = size
	}
}

func NewBlock(dstPath string, options ...BlockOption) (*Block, error) {
	data, err := os.Create(path.Join(dstPath, entity.NameData))
	if err != nil {
		return nil, fmt.Errorf("create data: %w", err)
	}

	block := &Block{
		path:         dstPath,
		block:        data,
		blockWriter:  NewWriteCounter(data),
		index:        NewBloom(),
		chunk:        NewChunk(data),
		maxChunkSize: 256 * 1024 * 1024, // 256MB
		buf:          make([]byte, 0, 4*1024*1024),
	}

	for _, option := range options {
		option(block)
	}

	return block, nil
}

func (b *Block) Close() error {
	b.wmu.Lock()
	defer b.wmu.Unlock()

	b.rmu.Lock()
	defer b.rmu.Unlock()

	if b.closed {
		return fmt.Errorf("block already closed")
	}

	b.closed = true

	if err := b.closeChunk(); err != nil {
		return fmt.Errorf("close last chunk: %w", err)
	}

	if err := b.block.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	index, err := os.Create(path.Join(b.path, entity.NameIndex))
	if err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	defer index.Close()

	if err := b.index.Write(index); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	return nil
}

func (b *Block) Search(ctx context.Context, target uuid.UUID) (*pb.Event, error) {
	b.rmu.RLock()
	defer b.rmu.RUnlock()

	if b.closed {
		return nil, fmt.Errorf("block already closed")
	}

	pointers := b.index.Search(target)
	if len(pointers) == 0 {
		return nil, entity.ErrNotFound
	}

	for _, pointer := range pointers {
		event, err := b.search(ctx, target, pointer)
		if errors.Is(err, entity.ErrNotFound) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("search: %w", err)
		}

		return event, nil
	}

	return nil, entity.ErrNotFound
}

func (b *Block) WritedSize() int {
	return b.size
}

func (b *Block) CompressedSize() int {
	return b.blockWriter.Size()
}

func (b *Block) Path() string {
	return b.path
}

func (b *Block) search(ctx context.Context, target uuid.UUID, pointer IndexPointer) (*pb.Event, error) {
	f, err := os.OpenFile(path.Join(b.path, entity.NameData), os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open data: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(pointer.ChunkOffset, 0); err != nil {
		return nil, fmt.Errorf("seek data: %w", err)
	}

	var r io.Reader = f
	if pointer.ChunkSize != -1 {
		r = io.LimitReader(f, pointer.ChunkSize)
	}

	event, err := SearchChunk(ctx, r, target)
	if err != nil {
		return nil, fmt.Errorf("search chunk: %w", err)
	}

	return event, nil
}

func (b *Block) Write(event *pb.Event) (int, error) {
	b.wmu.Lock()
	defer b.wmu.Unlock()

	if b.closed {
		return 0, fmt.Errorf("block alredy closed")
	}

	if b.chunk.WritedSize() > b.maxChunkSize {
		if err := b.closeChunk(); err != nil {
			return 0, fmt.Errorf("close chunk: %w", err)
		}

		b.chunk = NewChunk(b.blockWriter)
	}

	data, err := (proto.MarshalOptions{}).MarshalAppend(b.buf, event)
	if err != nil {
		return 0, fmt.Errorf("marshal: %w", err)
	}
	size := binary.LittleEndian.AppendUint64(nil, uint64(len(data)))

	sizeSize, err := b.chunk.Write(size)
	if err != nil {
		return 0, fmt.Errorf("write size: %w", err)
	}
	dataSize, err := b.chunk.Write(data)
	if err != nil {
		return 0, fmt.Errorf("write data: %w", err)
	}

	b.size += sizeSize + dataSize

	b.index.ChunkAppend(uuid.UUID(event.Id))

	return sizeSize + dataSize, nil
}

func (b *Block) closeChunk() error {
	if err := b.chunk.Close(); err != nil {
		return fmt.Errorf("chunk close: %w", err)
	}

	b.index.ChunkClose(b.chunk.CompressedSize())

	return nil
}
