package files3

import (
	"errors"
	"io"
)

type writeAt struct {
	w io.Writer
}

func newWriteAt(w io.Writer) *writeAt {
	return &writeAt{
		w: w,
	}
}

func (w *writeAt) WriteAt(b []byte, _ int64) (n int, err error) {
	return w.w.Write(b)
}

type multiWriter struct {
	w      io.Writer
	closes []func() error
}

func newMultiWriter(writers ...io.WriteCloser) *multiWriter {
	ws := make([]io.Writer, 0, len(writers))
	closes := make([]func() error, 0, len(writers))
	for _, w := range writers {
		ws = append(ws, w)
		closes = append(closes, w.Close)
	}

	return &multiWriter{
		w:      io.MultiWriter(ws...),
		closes: closes,
	}
}

func (m *multiWriter) Write(p []byte) (int, error) {
	return m.w.Write(p)
}

func (m *multiWriter) Close() error {
	var errs []error
	for _, close := range m.closes {
		errs = append(errs, close())
	}

	return errors.Join(errs...)
}

type closeWriter struct {
	w     io.WriteCloser
	after func() error
}

func (c *closeWriter) Write(p []byte) (int, error) {
	return c.w.Write(p)
}

func (c *closeWriter) Close() error {
	return errors.Join(c.w.Close(), c.after())
}
