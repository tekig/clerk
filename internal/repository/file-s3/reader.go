package files3

import (
	"io"
)

type limitedReader struct {
	r     io.Reader
	close func() error
}

func newLimitedReader(r io.ReadCloser, n int) io.ReadCloser {
	return &limitedReader{
		r:     io.LimitReader(r, int64(n)),
		close: r.Close,
	}
}

func (r *limitedReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *limitedReader) Close() error {
	return r.close()
}
