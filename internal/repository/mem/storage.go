package mem

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"slices"
	"sync"

	"github.com/tekig/clerk/internal/entity"
)

// Storage keeps blocks in memory. Written data is readable before the writer is closed,
// like the local copy of a block in the S3 storage.
type Storage struct {
	mu    sync.Mutex
	files map[string][]byte
}

func NewStorage() *Storage {
	return &Storage{
		files: make(map[string][]byte),
	}
}

func (s *Storage) Blocks(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var blocks []string
	for name := range s.files {
		block := path.Dir(name)
		if !slices.Contains(blocks, block) {
			blocks = append(blocks, block)
		}
	}
	slices.Sort(blocks)

	return blocks, nil
}

func (s *Storage) Read(ctx context.Context, block, name string) (io.ReadCloser, error) {
	return s.ReadRange(ctx, block, name, 0, -1)
}

func (s *Storage) ReadRange(ctx context.Context, block, name string, offset, size int) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.files[path.Join(block, name)]
	if !ok {
		return nil, fmt.Errorf("file `%s/%s`: %w", block, name, entity.ErrNotFound)
	}

	if offset > len(data) {
		offset = len(data)
	}
	data = data[offset:]
	if size != -1 && size < len(data) {
		data = data[:size]
	}

	return io.NopCloser(bytes.NewReader(bytes.Clone(data))), nil
}

func (s *Storage) Write(ctx context.Context, block, name string) (io.WriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := path.Join(block, name)
	s.files[key] = nil

	return &storageWriter{s: s, key: key}, nil
}

type storageWriter struct {
	s   *Storage
	key string
}

func (w *storageWriter) Write(p []byte) (int, error) {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()

	w.s.files[w.key] = append(w.s.files[w.key], p...)

	return len(p), nil
}

func (w *storageWriter) Close() error {
	return nil
}
