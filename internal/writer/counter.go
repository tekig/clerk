package writer

import "io"

type Counter[T io.Writer] struct {
	dst  T
	size int
}

func NewCounter[T io.Writer](dst T) *Counter[T] {
	return &Counter[T]{
		dst: dst,
	}
}

func (c *Counter[T]) Write(p []byte) (int, error) {
	n, err := c.dst.Write(p)

	c.size += n

	return n, err
}

func (c *Counter[T]) Origin() T {
	return c.dst
}

func (c *Counter[T]) Size() int {
	return c.size
}

func (c *Counter[T]) Reset() {
	c.size = 0
}
