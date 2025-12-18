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
	return &Iter{
		d:   d,
		Cmd: copyGetStatesCmd(cmd),
	}
}

// Next advances the iterator to the next state
// It returns true if there is a next state, false otherwise
// It is expected to call Next repeatedly until it returns false
func (it *Iter) Next() bool {
	if it.err != nil {
		return false
	}

	if it.res == nil {
		return it.more()
	}
	if len(it.res.States) == 0 {
		return false
	}
	it.resIdx++
	if it.resIdx >= len(it.res.States) {
		if it.res.More {
			return it.more()
		}

		return false
	}

	it.Cmd.SinceRev = it.res.States[it.resIdx].Rev
	return true
}

func (it *Iter) more() bool {
	it.Cmd.Result = nil
	if err := it.d.GetStates(it.Cmd); err != nil {
		it.err = err
		return false
	}

	it.res = it.Cmd.MustResult()
	it.resIdx = 0
	if len(it.res.States) == 0 {
		return false
	}

	it.Cmd.SinceRev = it.res.States[it.resIdx].Rev
	return true
}

// State returns the current state in the iteration
// It is expected to call State only when Next() returned true
func (it *Iter) State() State {
	if it.err != nil {
		panic("BUG: State() must not be called if there is an error; i.e. Next() returned false")
	}

	return it.res.States[it.resIdx]
}

// Err returns the error encountered during iteration, if any
// It is expected to call Err only when Next() returned false
// The iterator cannot be used once Err returns a non-nil error
// Create a new iterator using Iter.Cmd.
func (it *Iter) Err() error {
	return it.err
}

// Wait returns when new states are available or the context is done
// It is expected to call Wait only when Next() returned false
// It is expected to call Next() again after Wait() returns
// You can optionally check ctx.Err() to see if the context was done to break the loop
func (it *Iter) Wait(ctx context.Context) {
	if it.err != nil {
		panic("BUG: Wait() must not be called if there is an error; i.e. Next() returned false")
	}
	if it.res == nil || it.res.More {
		panic("BUG: Wait() must be called only when iterator reached the log head and there is no more states immediately available")
	}

	t := time.NewTimer(time.Millisecond * 100)
	t.Stop()

	for {
		if it.getStates() {
			return
		}

		t.Reset(time.Millisecond * 100)
		select {
		case <-t.C:
			t.Stop()
			continue
		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}

func (it *Iter) getStates() bool {
	it.Cmd.Result = nil

	if cd, ok := it.d.(*cacheDriver); ok {
		return it.getStatesCacheDriver(cd)
	}

	if err := it.d.GetStates(it.Cmd); err != nil {
		it.err = err
		return true
	}
	if res := it.Cmd.MustResult(); len(res.States) > 0 {
		it.res = res
		it.resIdx = -1
		return true
	}

	return false
}

func (it *Iter) getStatesCacheDriver(cd *cacheDriver) bool {
	minRev, maxRev, ok := cd.getStatesFromLog(it.Cmd)
	if ok {
		if res := it.Cmd.MustResult(); len(res.States) > 0 {
			it.res = res
			it.resIdx = -1
			return true
		}

		it.Cmd.SinceRev = maxRev
		return false
	}

	if err := cd.d.GetStates(it.Cmd); err != nil {
		it.err = err
		return true
	}
	if res := it.Cmd.MustResult(); len(res.States) > 0 {
		it.res = res
		it.resIdx = -1
		return true
	}

	if it.Cmd.SinceRev < minRev {
		it.Cmd.SinceRev = minRev
	}

	return false
}

func copyGetStatesCmd(cmd *GetStatesCommand) *GetStatesCommand {
	copyCmd := &GetStatesCommand{
		SinceRev:   cmd.SinceRev,
		SinceTime:  cmd.SinceTime,
		Labels:     copyORLabels(cmd.Labels),
		LatestOnly: cmd.LatestOnly,
		Limit:      cmd.Limit,
	}
	copyCmd.Prepare()

	return copyCmd
}

func copyORLabels(orLabels []map[string]string) []map[string]string {
	resORLabels := make([]map[string]string, 0, len(orLabels))

	for _, labels := range orLabels {
		copyLabels := make(map[string]string)
		for k, v := range labels {
			copyLabels[k] = v
		}
		resORLabels = append(resORLabels, copyLabels)
	}

	return resORLabels
}
