package uuid

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UUID [16]byte

func New() UUID {
	return UUID(uuid.Must(uuid.NewV7()))
}

func (id UUID) String() string {
	return uuid.UUID(id).String()
}

func (id UUID) Time() time.Time {
	if (id[6] & 0xF0) != 0x70 {
		return time.Time{}
	}

	var t int64

	t |= int64(id[0]) << 40
	t |= int64(id[1]) << 32
	t |= int64(id[2]) << 24
	t |= int64(id[3]) << 16
	t |= int64(id[4]) << 8
	t |= int64(id[5])

	return time.UnixMilli(t)
}

func FromBytes(raw []byte) (UUID, error) {
	if len(raw) != 16 {
		return UUID{}, fmt.Errorf("uuid size want=16, got=%d", len(raw))
	}

	return UUID(raw), nil
}

func FromString(raw string) (UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return UUID{}, fmt.Errorf("parse: %w", err)
	}

	return UUID(id), nil
}
