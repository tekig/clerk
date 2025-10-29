package bytes

import "io"

func Resize(b []byte, l int) []byte {
	if cap(b) >= l {
		return b[:l]
	}

	e := make([]byte, l-cap(b))
	b = append(b, e...)

	return b[:l]
}

func ReadAll(r io.Reader, b []byte) ([]byte, error) {
	for {
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}

		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
	}
}
