package memdriver

import (
	"github.com/makasim/flowstate"
)

type Watcher struct {
	sinceRev    int64
	sinceLatest bool
	labels      map[string]string

	watchCh  chan *flowstate.StateCtx
	changeCh chan int64
	closeCh  chan struct{}
	l        *Log
}

func (w *Watcher) Watch() <-chan *flowstate.StateCtx {
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

func (w *Watcher) listen() {
	var states []*flowstate.StateCtx

	if w.sinceLatest {
		w.l.Lock()
		_, sinceRev := w.l.LatestByLabels(w.labels)
		w.sinceRev = sinceRev - 1
		w.l.Unlock()
	}

skip:
	for {
		select {
		case <-w.changeCh:
			w.l.Lock()
			states, w.sinceRev = w.l.Entries(w.sinceRev, 10)
			w.l.Unlock()

			if len(states) == 0 {
				continue skip
			}

		next:
			for _, t := range states {
				for k, v := range w.labels {
					if t.Committed.Labels[k] != v {
						continue next
					}
				}

				select {
				case w.watchCh <- t:
					continue next
				case <-w.closeCh:
					return
				}
			}
		case <-w.closeCh:
			return
		}
	}
}
