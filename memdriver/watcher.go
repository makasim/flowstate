package memdriver

import "github.com/makasim/flowstate"

type Watcher struct {
	watchCh chan []*flowstate.TaskCtx
	closeCh chan struct{}
}

func (w *Watcher) Watch() chan []*flowstate.TaskCtx {
	return w.watchCh
}

func (w *Watcher) Close() {
	close(w.closeCh)
}
