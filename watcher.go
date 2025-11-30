package flowstate

import (
	"sync/atomic"
)

type Watcher struct {
	e   *Engine
	cmd *GetStatesCommand

	watchCh  chan State
	closeCh  chan struct{}
	closedCh chan struct{}
	closed   atomic.Bool
}

func newWatcher(e *Engine, cmd *GetStatesCommand) *Watcher {
	copyCmd := &GetStatesCommand{
		SinceRev:   cmd.SinceRev,
		SinceTime:  cmd.SinceTime,
		Labels:     copyORLabels(cmd.Labels),
		LatestOnly: cmd.LatestOnly,
		Limit:      cmd.Limit,
	}
	copyCmd.Prepare()

	w := &Watcher{
		e:   e,
		cmd: copyCmd,

		watchCh:  make(chan State, 1),
		closeCh:  make(chan struct{}),
		closedCh: make(chan struct{}),
	}

	go w.doWatch()

	return w
}

func (w *Watcher) Next() <-chan State {
	return w.watchCh
}

func (w *Watcher) Close() {
	if !w.closed.CompareAndSwap(false, true) {
		return
	}

	w.e.maxRevCond.L.Lock()
	close(w.closeCh)
	w.e.maxRevCond.Broadcast()
	w.e.maxRevCond.L.Unlock()

	<-w.closedCh
}

func (w *Watcher) doWatch() {
	defer close(w.closedCh)

	for {
		select {
		case <-w.e.doneCh:
			w.Close()
			return
			//panic("BUG: watchers should be closed before engine is closed")
		case <-w.closeCh:
			return
		default:
		}

		currMaxRev := w.e.maxRev.Load()
		w.stream()
		w.wait(currMaxRev)
	}
}

func (w *Watcher) stream() {
	for {
		getManyCmd := w.cmd

		getManyCmd.Result = nil
		if err := w.e.Do(getManyCmd); err != nil {
			w.e.l.Error("engine: do: get many states: %s", "error", err)
			w.Close()
			return
		}

		res := getManyCmd.MustResult()
		if len(res.States) == 0 {
			return
		}

		for i := range res.States {
			select {
			case w.watchCh <- res.States[i]:
				getManyCmd.SinceRev = res.States[i].Rev
			case <-w.closeCh:
				return
			}
		}

		if res.More {
			continue
		}

		return
	}
}

func (w *Watcher) wait(currMaxRev int64) {
	for {
		// fast path - max rev already advanced
		if currMaxRev < w.e.maxRev.Load() {
			return
		}

		w.e.maxRevCond.L.Lock()

		select {
		case <-w.closeCh:
			w.e.maxRevCond.L.Unlock()
			return
		default:
		}

		w.e.maxRevCond.Wait()
		w.e.maxRevCond.L.Unlock()

		select {
		case <-w.closeCh:
			return
		default:
		}
	}
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
