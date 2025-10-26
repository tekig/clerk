package uuid

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUUID7(t *testing.T) {
	now := time.Now()

	id := New()
	if id.Time() != now && id.Time().Sub(now) > time.Millisecond {
		t.Errorf("expected time %v, got %v, delta %v", now, id.Time(), now.Sub(id.Time()))
	}
}

func TestSupportUUID4(t *testing.T) {
	uuid4 := uuid.Must(uuid.NewRandom())

	if UUID(uuid4).Time() != (time.Time{}) {
		t.Errorf("expected zero time for UUIDv4, got %v", UUID(uuid4).Time())
	}
}
