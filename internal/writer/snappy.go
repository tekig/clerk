package writer

import (
	"io"

	"github.com/golang/snappy"
)

type Snappy[T io.Writer] struct {
	dst  T
	snap *snappy.Writer
}

func NewSnappy[T io.Writer](dst T) *Snappy[T] {
	return &Snappy[T]{
		dst:  dst,
		snap: snappy.NewBufferedWriter(dst),
	}
}

func (s *Snappy[T]) Write(b []byte) (int, error) {
	return s.snap.Write(b)
}

func (s *Snappy[T]) Close() error {
	return s.snap.Close()
}

func (s *Snappy[T]) Flush() error {
	return s.snap.Flush()
}

// Set header snappy for decode
func (s *Snappy[T]) Mark() {
	s.snap.Reset(s.dst)
}

func (s *Snappy[T]) Origin() T {
	return s.dst
}
