package writer

import (
	"bytes"
	"io"
	"testing"

	"github.com/golang/snappy"
)

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error {
	return nil
}

func TestCounter(t *testing.T) {
	var buf bytes.Buffer
	c := NewCounter(&buf)

	c.Write([]byte("hello "))
	c.Write([]byte("world"))

	if c.Size() != len("hello world") {
		t.Errorf("size = %d, want %d", c.Size(), len("hello world"))
	}
	if c.Origin() != &buf {
		t.Errorf("origin is not the destination writer")
	}
}

func TestSnappy_Mark(t *testing.T) {
	var buf bytes.Buffer
	compressed := NewCounter(nopCloser{&buf})
	s := NewSnappy(compressed)

	// Every mark starts a new stream that can be decoded from its offset
	parts := []string{"first chunk", "second chunk"}
	var offsets []int
	for _, p := range parts {
		offsets = append(offsets, compressed.Size())
		if _, err := s.Write([]byte(p)); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := s.Flush(); err != nil {
			t.Fatalf("flush: %v", err)
		}
		s.Mark()
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	for i, offset := range offsets {
		end := buf.Len()
		if i+1 < len(offsets) {
			end = offsets[i+1]
		}

		got, err := io.ReadAll(snappy.NewReader(bytes.NewReader(buf.Bytes()[offset:end])))
		if err != nil {
			t.Fatalf("read chunk #%d: %v", i, err)
		}
		if string(got) != parts[i] {
			t.Errorf("chunk #%d = %q, want %q", i, got, parts[i])
		}
	}

	// The whole data is also readable as one stream
	got, err := io.ReadAll(snappy.NewReader(bytes.NewReader(buf.Bytes())))
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if string(got) != parts[0]+parts[1] {
		t.Errorf("all = %q", got)
	}
}
