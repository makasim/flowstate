package memdriver

import "github.com/makasim/flowstate"

type Watcher struct {
	since  int64
	labels map[string]string

	watchCh  chan *flowstate.TaskCtx
	changeCh chan int64
	closeCh  chan struct{}
	l        *Log
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

func (w *Watcher) listen() {
	var tasks []*flowstate.TaskCtx

skip:
	for {
		select {
		case <-w.changeCh:
			w.l.Lock()
			tasks, w.since = w.l.Entries(w.since, 10)
			w.l.Unlock()

			if len(tasks) == 0 {
				continue skip
			}

		next:
			for _, t := range tasks {
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
