package xtime

import (
	"testing"
	"time"
)

func TestTicker(t *testing.T) {
	ticker := NewTicker(time.Millisecond)

	select {
	case <-ticker.C():
	case <-time.After(time.Second):
		t.Fatalf("no tick")
	}

	ticker.Close()

	// C is closed after Close, so `for range` loops are finished
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-ticker.C():
			if !ok {
				return
			}
		case <-deadline:
			t.Fatalf("channel is not closed")
		}
	}
}
