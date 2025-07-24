package memdriver

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Driver = &Driver{}

type Driver struct {
	stateLog        *stateLog
	dataLog         *dataLog
	delayedStateLog *delayedStateLog

	l *slog.Logger
}

func New(l *slog.Logger) *Driver {
	d := &Driver{
		stateLog:        &stateLog{},
		dataLog:         &dataLog{},
		delayedStateLog: &delayedStateLog{},

		l: l,
	}

	return d
}

func (d *Driver) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Driver) GetData(cmd *flowstate.GetDataCommand) error {
	data, err := d.dataLog.get(cmd.Data.ID, cmd.Data.Rev)
	if err != nil {
		return err
	}

	data.CopyTo(cmd.Data)
	return nil
}

func (d *Driver) StoreData(cmd *flowstate.AttachDataCommand) error {
	d.dataLog.append(cmd.Data)
	return nil
}

func (d *Driver) GetStateByID(cmd *flowstate.GetStateByIDCommand) error {
	if cmd.Rev == 0 {
		stateCtx, _ := d.stateLog.GetLatestByID(cmd.ID)
		if stateCtx == nil {
			return fmt.Errorf("%w; id=%s", flowstate.ErrNotFound, cmd.ID)
		}
		stateCtx.CopyTo(cmd.StateCtx)
		return nil
	}

	stateCtx := d.stateLog.GetByIDAndRev(cmd.ID, cmd.Rev)
	if stateCtx == nil {
		return fmt.Errorf("%w; id=%s rev=%d", flowstate.ErrNotFound, cmd.ID, cmd.Rev)
	}
	stateCtx.CopyTo(cmd.StateCtx)

	return nil
}

func (d *Driver) GetStateByLabels(cmd *flowstate.GetStateByLabelsCommand) error {
	stateCtx, _ := d.stateLog.GetLatestByLabels([]map[string]string{cmd.Labels})
	if stateCtx == nil {
		return fmt.Errorf("%w; labels=%v", flowstate.ErrNotFound, cmd.Labels)
	}
	stateCtx.CopyTo(cmd.StateCtx)

	return nil
}

func (d *Driver) GetStates(cmd *flowstate.GetStatesCommand) error {
	states := make([]flowstate.State, 0, cmd.Limit)
	limit := cmd.Limit + 1

	sinceRev := cmd.SinceRev
	if sinceRev == -1 {
		d.stateLog.Lock()
		var stateCtx *flowstate.StateCtx
		stateCtx, sinceRev = d.stateLog.GetLatestByLabels(cmd.Labels)
		if stateCtx != nil {
			states = append(states, stateCtx.Committed)
		}
		d.stateLog.Unlock()
	}

	d.stateLog.Lock()
	untilRev := d.stateLog.rev
	d.stateLog.Unlock()

	for {
		var logStates []*flowstate.StateCtx

		d.stateLog.Lock()
		logStates, sinceRev = d.stateLog.Entries(sinceRev, limit)
		d.stateLog.Unlock()

		if len(logStates) == 0 {
			cmd.Result = &flowstate.GetStatesResult{
				States: states,
			}
			return nil
		}

		for _, s := range logStates {
			if !cmd.SinceTime.IsZero() && s.Committed.CommittedAt.UnixMilli() < cmd.SinceTime.UnixMilli() {
				continue
			}
			if !matchLabels(s.Committed, cmd.Labels) {
				continue
			}

			if cmd.LatestOnly {
				states = filterStatesWithID(states, s.Committed.ID)
			}
			states = append(states, s.Committed)
		}

		if len(states) >= cmd.Limit {
			cmd.Result = &flowstate.GetStatesResult{
				States: states[:cmd.Limit],
				More:   len(states) > cmd.Limit,
			}
			return nil
		} else if sinceRev >= untilRev {
			cmd.Result = &flowstate.GetStatesResult{
				States: states,
				More:   false,
			}
			return nil
		}
	}
}

func (d *Driver) Delay(cmd *flowstate.DelayCommand) error {
	d.delayedStateLog.Append(*cmd.Result)

	return nil
}

func (d *Driver) GetDelayedStates(cmd *flowstate.GetDelayedStatesCommand) error {
	delayedStates := d.delayedStateLog.Get(cmd.Since, cmd.Until, cmd.Offset, cmd.Limit+1)

	more := false
	if len(delayedStates) > cmd.Limit {
		more = true
		delayedStates = delayedStates[:cmd.Limit]
	}

	cmd.Result = &flowstate.GetDelayedStatesResult{
		States: delayedStates,
		More:   more,
	}
	return nil
}

func (d *Driver) Commit(cmd *flowstate.CommitCommand) error {
	d.stateLog.Lock()
	defer d.stateLog.Unlock()
	defer d.stateLog.Rollback()

	for _, subCmd0 := range cmd.Commands {
		if err := flowstate.DoCommitSubCommand(d, subCmd0); err != nil {
			return fmt.Errorf("%T: do: %w", subCmd0, err)
		}

		subCmd, ok := subCmd0.(flowstate.CommittableCommand)
		if !ok {
			continue
		}

		stateCtx := subCmd.CommittableStateCtx()
		if stateCtx.Current.ID == `` {
			return fmt.Errorf("state id empty")
		}

		if _, rev := d.stateLog.GetLatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
			return &flowstate.ErrRevMismatch{IDS: []flowstate.StateID{stateCtx.Current.ID}}
		}

		d.stateLog.Append(stateCtx)
	}

	d.stateLog.Commit()

	return nil
}

func filterStatesWithID(states []flowstate.State, id flowstate.StateID) []flowstate.State {
	n := 0
	for _, state := range states {
		if state.ID != id {
			states[n] = state
			n++
		}
	}

	return states[:n]
}

type dataLog struct {
	sync.Mutex
	rev     int64
	entries []*flowstate.Data
}

func (l *dataLog) append(data *flowstate.Data) {
	l.Lock()
	defer l.Unlock()

	l.rev++
	data.Rev = l.rev
	l.entries = append(l.entries, data.CopyTo(&flowstate.Data{}))
}

func (l *dataLog) get(id flowstate.DataID, rev int64) (*flowstate.Data, error) {
	l.Lock()
	defer l.Unlock()

	for _, data := range l.entries {
		if data.ID == id && data.Rev == rev {
			return data, nil
		}
	}

	return nil, fmt.Errorf("data not found")
}

type stateLog struct {
	sync.Mutex
	rev     int64
	entries []*flowstate.StateCtx

	changes []*flowstate.StateCtx

	listeners []chan int64
}

func (l *stateLog) Append(stateCtx *flowstate.StateCtx) {
	committedT, _ := l.GetLatestByID(stateCtx.Current.ID)
	if committedT == nil {
		committedT = &flowstate.StateCtx{}
	}

	stateCtx.CopyTo(committedT)
	committedT.Current.CommittedAt = time.UnixMilli(time.Now().UnixMilli())
	committedT.Current.CopyTo(&committedT.Committed)
	committedT.Transitions = committedT.Transitions[:0]

	l.rev++
	committedT.Committed.Rev = l.rev
	committedT.Current.Rev = l.rev

	l.changes = append(l.changes, committedT)

	// todo: find a better place for this
	committedT.Committed.CopyTo(&stateCtx.Current)
	committedT.Committed.CopyTo(&stateCtx.Committed)
	stateCtx.Transitions = stateCtx.Transitions[:0]
}

func (l *stateLog) Commit() {
	slices.CompactFunc(l.changes, func(l, r *flowstate.StateCtx) bool {
		return l.Committed.ID == r.Committed.ID
	})

	slices.SortFunc(l.changes, func(l, r *flowstate.StateCtx) int {
		if l.Committed.Rev < r.Committed.Rev {
			return -1
		}

		return 1
	})

	var rev int64
	for _, stateCtx := range l.changes {
		rev = stateCtx.Current.Rev

		l.entries = append(l.entries, stateCtx)
	}

	l.changes = l.changes[:0]

	for _, ch := range l.listeners {
		select {
		case ch <- rev:
		case <-ch:
			ch <- rev
		}
	}
}

func (l *stateLog) Rollback() {
	l.changes = l.changes[:0]
}

func (l *stateLog) GetLatestByID(id flowstate.StateID) (*flowstate.StateCtx, int64) {
	var since int64
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == id {
			since = l.entries[i].Committed.Rev
			return l.entries[i].CopyTo(&flowstate.StateCtx{}), since
		}
	}

	return nil, since
}

func (l *stateLog) GetByIDAndRev(id flowstate.StateID, rev int64) *flowstate.StateCtx {
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == id && l.entries[i].Committed.Rev == rev {
			return l.entries[i].CopyTo(&flowstate.StateCtx{})
		}
	}

	return nil
}

func (l *stateLog) GetLatestByLabels(orLabels []map[string]string) (*flowstate.StateCtx, int64) {
	var nextSinceRev int64
next:
	for i := len(l.entries) - 1; i >= 0; i-- {
		if !matchLabels(l.entries[i].Committed, orLabels) {
			continue next
		}

		nextSinceRev = l.entries[i].Committed.Rev
		return l.entries[i].CopyTo(&flowstate.StateCtx{}), nextSinceRev
	}

	return nil, nextSinceRev
}

func (l *stateLog) Entries(since int64, limit int) ([]*flowstate.StateCtx, int64) {
	if limit == 0 {
		return nil, since
	}

	var entries []*flowstate.StateCtx
	for i := 0; i < len(l.entries); i++ {
		if l.entries[i].Committed.Rev <= since {
			continue
		}

		to := l.entries[i].CopyTo(&flowstate.StateCtx{})
		since = to.Committed.Rev

		entries = append(entries, to)
		if len(entries) == limit {
			break
		}
	}

	return entries, since
}

func (l *stateLog) SubscribeCommit(notifyCh chan int64) error {
	if cap(notifyCh) == 0 {
		return fmt.Errorf("notify channel is not buffered")
	}

	l.Lock()
	defer l.Unlock()

	l.listeners = append(l.listeners, notifyCh)
	return nil
}

func (l *stateLog) UnsubscribeCommit(notifyCh chan int64) {
	l.Lock()
	defer l.Unlock()

	for i, ch := range l.listeners {
		if ch == notifyCh {
			l.listeners = append(l.listeners[:i], l.listeners[i+1:]...)
			return
		}
	}
}

func matchLabels(state flowstate.State, orLabels []map[string]string) bool {
	if len(orLabels) == 0 {
		return true
	}

next:
	for _, labels := range orLabels {
		if len(labels) == 0 {
			continue
		}
		if len(labels) > len(state.Labels) {
			continue
		}

		for k, v := range labels {
			if state.Labels[k] != v {
				continue next
			}
		}

		return true
	}

	return false
}

type delayedStateLog struct {
	sync.Mutex
	offset  int64
	entries []flowstate.DelayedState
}

func (l *delayedStateLog) Append(delayedState flowstate.DelayedState) {
	l.Lock()
	defer l.Unlock()

	l.offset++
	delayedState.Offset = l.offset
	l.entries = append(l.entries, delayedState)
}

func (l *delayedStateLog) Get(since, until time.Time, offset int64, limit int) []flowstate.DelayedState {
	l.Lock()
	defer l.Unlock()

	var result []flowstate.DelayedState
	for _, delayedState := range l.entries {
		if delayedState.ExecuteAt.Before(since) {
			continue
		}
		if delayedState.ExecuteAt.After(until) {
			continue
		}
		if delayedState.Offset <= offset {
			continue
		}

		result = append(result, delayedState)
		if len(result) >= limit {
			break
		}
	}

	return result
}
