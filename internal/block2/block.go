package block2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom"
	"github.com/golang/snappy"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"github.com/tekig/clerk/internal/writer"
)

type Block struct {
	dir          string
	blockWriter  *writer.Counter[*writer.Snappy[*writer.Counter[*os.File]]]
	indexWriter  *os.File
	currentIndex *pb.Index_Chunk
	prevSize     int
	count        int

	wmu    sync.Mutex   // mutex from write
	rmu    sync.RWMutex // mutex from read
	closed bool

	// Maximum time that data can be in the buffer.
	// If there are few events, the last event may be displayed as not found.
	maxBufDuration *time.Ticker
	maxChunkSize   int
}

type BlockOption func(*Block)

func MaxChunkSize(size int) BlockOption {
	return func(b *Block) {
		b.maxChunkSize = size
	}
}

func NewBlock(dir string, options ...BlockOption) (*Block, error) {
	fblock, err := os.Create(path.Join(dir, entity.NameData))
	if err != nil {
		return nil, fmt.Errorf("create block: %w", err)
	}

	findex, err := os.Create(path.Join(dir, entity.NameIndex))
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	block := &Block{
		dir: dir,
		blockWriter: writer.NewCounter(
			writer.NewSnappy(
				writer.NewCounter(fblock),
			),
		),
		indexWriter: findex,
		currentIndex: &pb.Index_Chunk{
			Mark: &pb.Index_Chunk_Mark{
				Size:   -1,
				Offset: 0,
			},
		},

		maxBufDuration: time.NewTicker(30 * time.Second),
		maxChunkSize:   64 * 1024 * 1024,
	}

	go func() {
		for range block.maxBufDuration.C {
			if block.wmu.TryLock() {
				if block.closed {
					block.wmu.Unlock()
					break
				}
				_ = block.blockWriter.Origin().Flush()
				block.wmu.Unlock()
			}
		}
	}()

	for _, o := range options {
		o(block)
	}

	return block, nil
}

func (b *Block) Write(event *pb.Event) error {
	b.wmu.Lock()
	defer b.wmu.Unlock()

	if b.closed {
		return fmt.Errorf("block alredy closed")
	}

	if b.blockWriter.Size()-b.prevSize > b.maxChunkSize {
		if err := b.nextChuck(); err != nil {
			return fmt.Errorf("next chuck: %w", err)
		}
	}

	if err := Encode(event, b.blockWriter); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	b.currentIndex.Ids = append(b.currentIndex.Ids, event.Id)
	b.count++

	return nil
}

func (b *Block) Search(ctx context.Context, target uuid.UUID) (*pb.Event, error) {
	b.rmu.RLock()
	defer b.rmu.RUnlock()

	if b.closed {
		return nil, fmt.Errorf("block already closed")
	}

	mark, err := b.indexSearch(target)
	if err != nil {
		return nil, fmt.Errorf("index search: %w", err)
	}

	block, err := os.Open(path.Join(b.dir, entity.NameData))
	if err != nil {
		return nil, fmt.Errorf("open block: %w", err)
	}
	defer block.Close()

	if _, err := block.Seek(mark.Offset, 0); err != nil {
		return nil, fmt.Errorf("seek block: %w", err)
	}

	var r io.Reader = block
	if mark.Size != -1 {
		r = io.LimitReader(block, mark.Size)
	}

	snap := snappy.NewReader(r)

	for {
		var event = &pb.Event{}
		if err := Decode(event, snap); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read event: %w", err)
		}
		if uuid.UUID(event.Id) == target {
			return event, nil
		}
	}

	return nil, fmt.Errorf("nothing found in the specified mark")
}

func (b *Block) indexSearch(target uuid.UUID) (*pb.Index_Chunk_Mark, error) {
	for _, id := range b.currentIndex.Ids {
		if uuid.UUID(id) == target {
			return b.currentIndex.Mark, nil
		}
	}

	idx, err := os.Open(path.Join(b.dir, entity.NameIndex))
	if err != nil {
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer idx.Close()

	for {
		var index = &pb.Index_Chunk{}
		if err := Decode(index, idx); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read index: %w", err)
		}
		for _, id := range index.Ids {
			if uuid.UUID(id) == target {
				return index.Mark, nil
			}
		}
	}

	return nil, entity.ErrNotFound
}

func (b *Block) WritedSize() int {
	return b.blockWriter.Size()
}

func (b *Block) CompressedSize() int {
	return b.blockWriter.Origin().Origin().Size()
}

func (b *Block) Path() string {
	return b.dir
}

func (b *Block) Close() error {
	b.wmu.Lock()
	defer b.wmu.Unlock()

	b.rmu.Lock()
	defer b.rmu.Unlock()

	if b.closed {
		return fmt.Errorf("block already clodes")
	}

	b.closed = true

	b.maxBufDuration.Stop()

	if err := b.nextChuck(); err != nil {
		return fmt.Errorf("next chuck: %w", err)
	}

	if err := b.blockWriter.Origin().Close(); err != nil {
		return fmt.Errorf("snappy close: %w", err)
	}

	if err := b.blockWriter.Origin().Origin().Origin().Close(); err != nil {
		return fmt.Errorf("file close: %w", err)
	}

	if err := b.indexWriter.Close(); err != nil {
		return fmt.Errorf("index close: %w", err)
	}

	if err := b.createBloom(); err != nil {
		return fmt.Errorf("create bloom: %w", err)
	}

	return nil
}

func (b *Block) createBloom() error {
	idx, err := os.Open(path.Join(b.dir, entity.NameIndex))
	if err != nil {
		return fmt.Errorf("open index: %w", err)
	}
	defer idx.Close()

	bl := bloom.NewWithEstimates(uint(b.count), 0.01)
	for {
		var index = &pb.Index_Chunk{}
		if err := Decode(index, idx); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read index: %w", err)
		}
		for _, id := range index.Ids {
			bl.Add(id)
		}
	}

	var buf = bytes.NewBuffer(nil)
	if _, err := bl.WriteTo(buf); err != nil {
		return fmt.Errorf("conv bloom: %w", err)
	}

	bloom, err := os.Create(path.Join(b.dir, entity.NameBloom))
	if err != nil {
		return fmt.Errorf("create bloom: %w", err)
	}
	defer bloom.Close()

	if err := Encode(&pb.Bloom{Bloom: buf.Bytes()}, bloom); err != nil {
		return fmt.Errorf("write bloom: %w", err)
	}

	return nil
}

func (b *Block) nextChuck() error {
	b.prevSize = b.blockWriter.Size()

	if err := b.blockWriter.Origin().Flush(); err != nil {
		return fmt.Errorf("flush block: %w", err)
	}

	b.blockWriter.Origin().Mark()

	b.currentIndex.Mark.Size = int64(b.blockWriter.Origin().Origin().Size()) - b.currentIndex.Mark.Offset

	if err := Encode(b.currentIndex, b.indexWriter); err != nil {
		return fmt.Errorf("write index: %w", err)
	}

	b.currentIndex = &pb.Index_Chunk{
		Mark: &pb.Index_Chunk_Mark{
			Size:   -1,
			Offset: int64(b.blockWriter.Origin().Origin().Size()),
		},
	}

	return nil
}
