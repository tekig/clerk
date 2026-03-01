package xsync

import "sync"

type ErrGroup struct {
	wg       sync.WaitGroup
	firstErr error
	once     sync.Once
}

func (g *ErrGroup) Go(fn func() error) {
	g.wg.Go(func() {
		if err := fn(); err != nil {
			g.once.Do(func() {
				g.firstErr = err
			})
		}
	})
}

func (g *ErrGroup) Wait() error {
	g.wg.Wait()

	return g.firstErr
}
