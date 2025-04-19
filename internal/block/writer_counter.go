package block

import "io"

type WriteCounter struct {
	dst  io.Writer
	size int
}

func NewWriteCounter(dst io.Writer) *WriteCounter {
	return &WriteCounter{
		dst: dst,
	}
}

func (w *WriteCounter) Size() int {
	return w.size
}

func (w *WriteCounter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)

	w.size += n

	return n, err
}
