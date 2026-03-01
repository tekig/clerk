package xtime

import "time"

type Ticker struct {
	stop   chan struct{}
	ticker *time.Ticker
	c      chan time.Time
}

func NewTicker(d time.Duration) *Ticker {
	t := &Ticker{
		ticker: time.NewTicker(d),
		stop:   make(chan struct{}),
		c:      make(chan time.Time),
	}

	go func() {
		func() {
			for {
				select {
				case tick := <-t.ticker.C:
					select {
					case t.c <- tick:
					case <-t.stop:
						return
					}
				case <-t.stop:
					return
				}
			}
		}()

		t.ticker.Stop()
		close(t.c)
	}()

	return t
}

// Channel for receiving ticks. The channel must be read using for or handled by ok.
//
// Example:
//
//	t := NewTimer(10 * time.Second)
//	for range t {
//	  do()
//	}
func (t *Ticker) C() <-chan time.Time {
	return t.c
}

// Close ticker and send close message chan to C()
func (t *Ticker) Close() {
	close(t.stop)
}
