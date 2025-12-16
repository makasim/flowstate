package flowstate

import (
	"context"
	"log/slog"
	"time"
)

type Watcher struct {
	e    *Engine
	iter *Iter

	watchCh  chan State
	ctx      context.Context
	cancel   context.CancelFunc
	closedCh chan struct{}
}

func NewWatcher(e *Engine, cmd *GetStatesCommand) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watcher{
		e:    e,
		iter: NewIter(e.d, cmd),

		watchCh:  make(chan State, 1),
		ctx:      ctx,
		cancel:   cancel,
		closedCh: make(chan struct{}),
	}

	go w.listen()

	return w
}

func (w *Watcher) Next() <-chan State {
	return w.watchCh
}

func (w *Watcher) Close() {
	w.cancel()
	<-w.closedCh
}

func (w *Watcher) listen() {
	defer close(w.closedCh)

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		for w.iter.Next() {
			select {
			case w.watchCh <- w.iter.State():
			case <-w.ctx.Done():
				return
			}
		}

		if w.iter.err != nil {
			w.e.l.Error("watcher: stream: iter: next", slog.String("err", w.iter.err.Error()))
			return
		}

		ctx, cancel := context.WithTimeout(w.ctx, time.Second*5)
		w.iter.Wait(ctx)
		cancel()
	}
}
