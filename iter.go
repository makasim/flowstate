package flowstate

import (
	"context"
	"time"
)

type Iter struct {
	d      Driver
	cmd    *GetStatesCommand
	res    *GetStatesResult
	resIdx int

	s   *State
	err error
}

func newIter(d Driver, cmd *GetStatesCommand) *Iter {
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
		cmd: copyCmd,
	}
}

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
	if w.resIdx >= len(w.res.States) {
		if w.res.More {
			return w.more()
		}

		return false
	}

	w.resIdx++
	w.res.States[w.resIdx].CopyTo(w.s)
	w.cmd.SinceRev = w.s.Rev
	return true
}

func (w *Iter) State() *State {
	if w.err != nil {
		panic("BUG: State() must not be called if there is an error; i.e. Next() returned false")
	}

	return w.s
}

func (w *Iter) Err() error {
	return w.err
}

func (w *Iter) Wait(ctx context.Context) {
	if w.err != nil {
		panic("BUG: Wait() must not be called if there is an error; i.e. Next() returned false")
	}
	if w.res == nil || (len(w.res.States) > 0 && w.res.More) {
		panic("BUG: Wait() must be called only when watcher reached the log head and there is no more states available")
	}

	for {
		if w.more() {
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
