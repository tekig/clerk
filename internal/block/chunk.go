package block

import (
	"fmt"
	"io"

	"github.com/golang/snappy"
)

type Chunk struct {
	writed   int
	counter  *WriteCounter
	compress *snappy.Writer
}

func NewChunk(w io.Writer) *Chunk {
	counter := NewWriteCounter(w)
	return &Chunk{
		counter:  counter,
		compress: snappy.NewBufferedWriter(counter),
	}
}

func (c *Chunk) Write(b []byte) (int, error) {
	n, err := c.compress.Write(b)
	if err != nil {
		return 0, fmt.Errorf("compress: %w", err)
	}

	c.writed += n

	return n, nil
}

func (c *Chunk) WritedSize() int {
	return c.writed
}

func (c *Chunk) CompressedSize() int {
	return c.counter.size
}

func (c *Chunk) Close() error {
	if err := c.compress.Close(); err != nil {
		return fmt.Errorf("compress close: %w", err)
	}

	return nil
}
