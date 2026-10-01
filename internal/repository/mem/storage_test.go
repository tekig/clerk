package mem

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/tekig/clerk/internal/entity"
)

func readAll(t *testing.T) func(r io.ReadCloser, err error) string {
	return func(r io.ReadCloser, err error) string {
		t.Helper()

		if err != nil {
			t.Fatalf("read: %v", err)
		}
		defer r.Close()

		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("read all: %v", err)
		}

		return string(data)
	}
}

func TestStorage(t *testing.T) {
	ctx := context.Background()
	s := NewStorage()

	w, err := s.Write(ctx, "b1", "data")
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := w.Write([]byte("hello ")); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Data is readable before close
	if got := readAll(t)(s.Read(ctx, "b1", "data")); got != "hello " {
		t.Errorf("read before close = %q", got)
	}

	if _, err := w.Write([]byte("world")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	tests := []struct {
		offset, size int
		want         string
	}{
		{0, -1, "hello world"},
		{6, -1, "world"},
		{0, 5, "hello"},
		{6, 100, "world"},
		{100, -1, ""},
	}
	for _, tt := range tests {
		if got := readAll(t)(s.ReadRange(ctx, "b1", "data", tt.offset, tt.size)); got != tt.want {
			t.Errorf("ReadRange(%d, %d) = %q, want %q", tt.offset, tt.size, got, tt.want)
		}
	}

	if _, err := s.Read(ctx, "b1", "index"); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("read missing file: err = %v, want %v", err, entity.ErrNotFound)
	}

	if _, err := s.Write(ctx, "b2", "data"); err != nil {
		t.Fatalf("write: %v", err)
	}
	blocks, err := s.Blocks(ctx)
	if err != nil {
		t.Fatalf("blocks: %v", err)
	}
	if !slices.Equal(blocks, []string{"b1", "b2"}) {
		t.Errorf("blocks = %v", blocks)
	}
}
