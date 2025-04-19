package awss3

import (
	"io"
)

type WriteAt struct {
	w io.Writer
}

func (w *WriteAt) WriteAt(b []byte, _ int64) (n int, err error) {
	return w.w.Write(b)
}
