package memdriver

import "github.com/makasim/flowstate"

type Watcher struct {
	watchCh  chan *flowstate.TaskCtx
	changeCh chan int64
	closeCh  chan struct{}
}

func (w *Watcher) Watch() <-chan *flowstate.TaskCtx {
	return w.watchCh
}

func (w *Watcher) Close() {
	close(w.closeCh)
}

func (w *Watcher) Change(rev int64) {
	select {
	case w.changeCh <- rev:
	case <-w.changeCh:
		w.changeCh <- rev
	}
}
