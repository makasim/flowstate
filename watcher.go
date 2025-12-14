package flowstate

import (
	"log"
	"time"
)

type Watcher struct {
	e *Engine

	cmd      *GetStatesCommand
	pollDur  time.Duration
	watchCh  chan State
	closeCh  chan struct{}
	closedCh chan struct{}
}

func NewWatcher(e *Engine, pollDur time.Duration, cmd *GetStatesCommand) *Watcher {
	copyCmd := &GetStatesCommand{
		SinceRev:   cmd.SinceRev,
		SinceTime:  cmd.SinceTime,
		Labels:     copyORLabels(cmd.Labels),
		LatestOnly: cmd.LatestOnly,
		Limit:      cmd.Limit,
	}
	copyCmd.Prepare()

	w := &Watcher{
		e:       e,
		cmd:     copyCmd,
		pollDur: pollDur,

		watchCh:  make(chan State, 1),
		closeCh:  make(chan struct{}),
		closedCh: make(chan struct{}),
	}

	go w.listen()

	return w
}

func (w *Watcher) Next() <-chan State {
	return w.watchCh
}

func (w *Watcher) Close() {
	close(w.closeCh)
	<-w.closedCh
}

func (w *Watcher) listen() {
	defer close(w.closedCh)

	t := time.NewTicker(w.pollDur)
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
	getManyCmd := w.cmd

	getManyCmd.Result = nil
	if err := w.e.Do(getManyCmd); err != nil {
		log.Printf("ERROR: WatchListener: get many states: %s", err)
		return false
	}

	res := getManyCmd.MustResult()
	if len(res.States) == 0 {
		return false
	}

	for i := range res.States {
		select {
		case w.watchCh <- res.States[i]:
			getManyCmd.SinceRev = res.States[i].Rev
		case <-w.closeCh:
			return false
		}
	}

	return res.More
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
