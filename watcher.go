package flowstate

import (
	"log"
	"time"
)

type Watcher struct {
	e Engine

	sinceRev   int64
	sinceTime  time.Time
	orLabels   []map[string]string
	latestOnly bool
	limit      int

	cmd     *GetStatesCommand
	watchCh chan State
	closeCh chan struct{}
}

func NewWatcher(e Engine, orLabels []map[string]string, sinceRev int64, sinceTime time.Time, latestOnly bool, limit int) *Watcher {
	w := &Watcher{
		e: e,

		sinceRev:   sinceRev,
		sinceTime:  sinceTime,
		orLabels:   copyORLabels(orLabels),
		latestOnly: latestOnly,
		limit:      limit,

		watchCh: make(chan State, 1),
		closeCh: make(chan struct{}),
	}

	go w.listen()

	return w
}

func (w *Watcher) Next() <-chan State {
	return w.watchCh
}

func (w *Watcher) Close() {
	close(w.closeCh)
}

func (w *Watcher) listen() {
	t := time.NewTicker(time.Millisecond * 100)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			for {
				if more := w.stream(); !more {
					break
				}
			}
		case <-w.closeCh:
			return
		}
	}
}

func (w *Watcher) stream() bool {
	states, more, err := w.e.GetStates(w.orLabels, w.sinceRev, w.sinceTime, w.latestOnly, w.limit)
	if err != nil {
		log.Printf("ERROR: WatchListener: get many states: %s", err)
		return false
	}

	if len(states) == 0 {
		return false
	}

	for i := range states {
		select {
		case w.watchCh <- states[i]:
			w.sinceRev = states[i].Rev
		case <-w.closeCh:
			return false
		}
	}

	return more
}

func copyORLabels(orLabels []map[string]string) []map[string]string {
	copyORLabels := make([]map[string]string, 0, len(orLabels))

	for _, labels := range orLabels {
		copyLabels := make(map[string]string)
		for k, v := range labels {
			copyLabels[k] = v
		}
		copyORLabels = append(copyORLabels, copyLabels)
	}

	return copyORLabels
}
