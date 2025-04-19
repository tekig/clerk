package uuid

import (
	"fmt"

	"github.com/google/uuid"
)

type UUID [16]byte

func New() UUID {
	return UUID(uuid.New())
}

func (id UUID) String() string {
	return uuid.UUID(id).String()
}

func Must(id UUID, err error) UUID {
	if err != nil {
		panic(fmt.Sprintf("must UUID: %s", err.Error()))
	}

	return id
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
