package block

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/golang/snappy"
	"github.com/tekig/clerk/internal/entity"
	"github.com/tekig/clerk/internal/pb"
	"github.com/tekig/clerk/internal/uuid"
	"google.golang.org/protobuf/proto"
)

func SearchChunk(ctx context.Context, r io.Reader, target uuid.UUID) (*pb.Event, error) {
	decode := snappy.NewReader(r)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var sizeRaw = make([]byte, 8)
		sizeN, err := ReadFull(decode, sizeRaw)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, entity.ErrNotFound
			}

			return nil, fmt.Errorf("read size: %w", err)
		}
		if sizeN != len(sizeRaw) {
			return nil, fmt.Errorf("read size: readed %d, should be %d", sizeN, len(sizeRaw))
		}

		var dataRaw = make([]byte, binary.LittleEndian.Uint64(sizeRaw))
		dataN, err := ReadFull(decode, dataRaw)
		if err != nil {
			return nil, fmt.Errorf("read data: %w", err)
		}
		if dataN != len(dataRaw) {
			return nil, fmt.Errorf("read data: readed %d, should be %d", sizeN, len(dataRaw))
		}

		var event = &pb.Event{}
		if err := proto.Unmarshal(dataRaw, event); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

		if uuid.UUID(event.Id) == target {
			return event, nil
		}
	}
}

func ReadFull(r io.Reader, b []byte) (retN int, err error) {
	for {
		n, err := r.Read(b)
		if err != nil {
			return retN, fmt.Errorf("read: %w", err)
		}

		retN += n

		if len(b) == n {
			return retN, nil
		}

		b = b[n:]
	}
}
