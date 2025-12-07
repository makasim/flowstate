package flowstate

import (
	"context"
	"sync/atomic"
	"time"
)

type Watcher struct {
	e   *Engine
	cmd *GetStatesCommand

	watchCh  chan State
	closeCh  chan struct{}
	closedCh chan struct{}
	closed   atomic.Bool

	Deadline time.Duration
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

// Watch calls the provided function f for each State gotten.
// If f returns false, the Watch function return, it could be called again to continue watching.
// State in f should not be stored or modified outside f.
// Use State.CopyTo or State.CopyToCtx to copy it if needed.
// The Watch returns if no state got in Watcher.Deadline duration with no error.
func (w *Watcher) Watch(f func(s *State) bool) error {
	t := time.NewTicker(w.Deadline)
	defer t.Stop()

	for {
		currMaxRev := w.e.maxRev.Load()
		w.stream()
		w.wait(currMaxRev)
	}
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

func (w *Watcher) stream2(f func(s *State) bool) {
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
			f(&res.States[i])
			getManyCmd.SinceRev = res.States[i].Rev
		}

		if res.More {
			continue
		}

		return
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

type watcher2 struct {
	d      *cacheDriver
	cmd    *GetStatesCommand
	res    *GetStatesResult
	resIdx int

	s   *State
	err error
}

func newWatcher2(d *cacheDriver, cmd *GetStatesCommand) *watcher2 {
	copyCmd := &GetStatesCommand{
		SinceRev:   cmd.SinceRev,
		SinceTime:  cmd.SinceTime,
		Labels:     copyORLabels(cmd.Labels),
		LatestOnly: cmd.LatestOnly,
		Limit:      cmd.Limit,
	}
	copyCmd.Prepare()

	return &watcher2{
		d:   d,
		cmd: copyCmd,
	}
}

func (w *watcher2) Next() bool {
	if w.err != nil {
		return false
	}

	if w.res == nil {
		return w.more()
	}
	if len(w.res.States) == 0 {
		return false
	}
	if w.resIdx >= len(w.res.States) {
		if w.res.More {
			w.cmd.SinceRev = w.res.States[len(w.res.States)-1].Rev
			return w.more()
		}

		return false
	}

	w.resIdx++
	return true
}

func (w *watcher2) State() State {
	if w.err != nil {
		panic("BUG: State() must not be called if there is an error; i.e. Next() returned false")
	}

	return w.res.States[w.resIdx]
}

func (w *watcher2) Err() error {
	return w.err
}

func (w *watcher2) Wait(ctx context.Context) {
	if w.err != nil {
		panic("BUG: Wait() must not be called if there is an error; i.e. Next() returned false")
	}
	if w.res == nil || (len(w.res.States) > 0 && w.res.More) {
		panic("BUG: Wait() must be called only when watcher reached the log head and there is no more states available")
	}

	for {
		w.d.m.Lock()
		minRev, maxRev, size, maxSize := w.d.minRev, w.d.maxRev, w.d.size, len(w.d.log)
		w.d.m.Unlock()

		if w.cmd.SinceRev < minRev {
			w.cmd.SinceRev = minRev
			if size == maxSize {
				// read from the middle
				w.cmd.SinceRev = (maxRev-minRev)/2 + minRev
			}
		}

		if w.more() {
			return
		}

		w.cmd.SinceRev = maxRev

		t := time.NewTimer(time.Millisecond * 100)
		select {
		case <-t.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (w *watcher2) more() bool {
	w.cmd.Result = nil
	if err := w.d.GetStates(w.cmd); err != nil {
		w.err = err
		return false
	}

	w.res = w.cmd.MustResult()
	w.resIdx = 0

	if len(w.res.States) == 0 {
		return false
	}

	return true
}

//
//func foo(w *watcher2) {
//
//	for {
//		for w.Next() {
//			s := w.State()
//			// handle state
//		}
//
//		if w.Err() != nil {
//			//handle error
//		}
//
//		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
//		w.Wait(ctx)
//	}
//
//}
