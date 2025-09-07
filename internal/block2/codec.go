package block2

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"
)

var poolBuf = sync.Pool{
	New: func() any {
		return make([]byte, 0, 4*1024*1024)
	},
}

func Encode(m proto.Message, w io.Writer) error {
	b := poolBuf.Get().([]byte)
	defer func() {
		// put in buf final slice
		poolBuf.Put(b[:0])
	}()

	b, err := (proto.MarshalOptions{}).MarshalAppend(b, m)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	size := binary.LittleEndian.AppendUint64(nil, uint64(len(b)))

	if _, err := w.Write(size); err != nil {
		return fmt.Errorf("write size: %w", err)
	}

	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	return nil
}

func resize(b []byte, l int) []byte {
	if len(b) >= l {
		return b[:l]
	}

	e := make([]byte, l-len(b))
	b = append(b, e...)

	return b
}

func Decode(m proto.Message, r io.Reader) error {
	b := poolBuf.Get().([]byte)
	defer func() {
		// put in buf final slice
		poolBuf.Put(b[:0])
	}()

	b = resize(b, 8)
	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("read size: %w", err)
	}

	b = resize(b, int(binary.LittleEndian.Uint64(b)))
	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("read data: %w", err)
	}

	if err := proto.Unmarshal(b, m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	return nil
}
