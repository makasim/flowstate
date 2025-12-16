package flowstate

import (
	"context"
	"time"
)

type Iter struct {
	Cmd *GetStatesCommand

	d      Driver
	res    *GetStatesResult
	resIdx int
	err    error
}

func NewIter(d Driver, cmd *GetStatesCommand) *Iter {
	copyCmd := &GetStatesCommand{
		SinceRev:   cmd.SinceRev,
		SinceTime:  cmd.SinceTime,
		Labels:     copyORLabels(cmd.Labels),
		LatestOnly: cmd.LatestOnly,
		Limit:      cmd.Limit,
	}
	copyCmd.Prepare()

	return &Iter{
		d:   d,
		Cmd: copyCmd,
	}
}

// Next advances the iterator to the next state
// It returns true if there is a next state, false otherwise
// It is expected to call Next repeatedly until it returns false
func (w *Iter) Next() bool {
	if w.err != nil {
		return false
	}

	if w.res == nil {
		return w.more()
	}
	if len(w.res.States) == 0 {
		return false
	}
	w.resIdx++
	if w.resIdx >= len(w.res.States) {
		if w.res.More {
			return w.more()
		}

		return false
	}

	w.Cmd.SinceRev = w.res.States[w.resIdx].Rev
	return true
}

// State returns the current state in the iteration
// It is expected to call State only when Next() returned true
func (w *Iter) State() State {
	if w.err != nil {
		panic("BUG: State() must not be called if there is an error; i.e. Next() returned false")
	}

	return w.res.States[w.resIdx]
}

// Err returns the error encountered during iteration, if any
// It is expected to call Err only when Next() returned false
// The iterator cannot be used once Err returns a non-nil error
// Create a new iterator using Iter.Cmd.
func (w *Iter) Err() error {
	return w.err
}

// Wait returns when new states are available or the context is done
// It is expected to call Wait only when Next() returned false
// It is expected to call Next() again after Wait() returned true
// You can optionally check ctx.Err() to see if the context was done to break the loop
func (w *Iter) Wait(ctx context.Context) {
	if w.err != nil {
		panic("BUG: Wait() must not be called if there is an error; i.e. Next() returned false")
	}
	if w.res == nil || w.res.More {
		panic("BUG: Wait() must be called only when watcher reached the log head and there is no more states immediately available")
	}

	for {
		w.Cmd.Result = nil
		if err := w.d.GetStates(w.Cmd); err != nil {
			w.err = err
			return
		}
		if res := w.Cmd.MustResult(); len(res.States) > 0 {
			w.res = res
			w.resIdx = -1
			return
		}

		t := time.NewTimer(time.Millisecond * 100)
		select {
		case <-t.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (w *Iter) more() bool {
	w.Cmd.Result = nil
	if err := w.d.GetStates(w.Cmd); err != nil {
		w.err = err
		return false
	}

	w.res = w.Cmd.MustResult()
	w.resIdx = 0
	if len(w.res.States) == 0 {
		return false
	}

	//if len(w.res.States) > 0 {
	//	log.Println(len(w.res.States), w.res.More, w.res.States[0].ID)
	//} else {
	//	log.Println(len(w.res.States), w.res.More)
	//}

	w.Cmd.SinceRev = w.res.States[w.resIdx].Rev
	return true
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
