package flowstate

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

var _ Driver = &cacheDriver{}

// A cacheDriver is a driver that keeps a head of states and updates them by worker from time to time.
// The head refresh can also be triggered by state commit commands.
type cacheDriver struct {
	d Driver
	l *slog.Logger

	m         sync.Mutex
	size      int   // current number of elements in the buffer (0 to len(log))
	idx       int   // next write position in the cycle buffer
	minRev    int64 // minimum revision available in the log
	maxRev    int64 // maximum revision available in the log
	log       []State
	committed []State
}

func NewCacheDriver(d Driver, maxSize int, l *slog.Logger) Driver {
	return newCacheDriver(d, maxSize, l)
}

// newCacheDriver creates a new cacheDriver.
// d is the underlying driver to wrap.
// maxSize is the maximum number of states to keep in the cache.
//
// Please note that the cacheDriver does not start the head refresh worker - it must be started by the caller.
func newCacheDriver(d Driver, maxSize int, l *slog.Logger) *cacheDriver {
	return &cacheDriver{
		d: d,
		l: l,

		log:       make([]State, maxSize),
		committed: make([]State, 0, 100),
		size:      0,
		idx:       0,
		minRev:    -1,
		maxRev:    0,
	}
}

// appendStateLocked appends a state to the cycle buffer log.
// Must be called with d.m held.
func (d *cacheDriver) appendStateLocked(s *State) {
	if s.Rev <= d.maxRev {
		panic("BUG: appendStateLocked: state revision must be greater than maxRev; client responsibility to ensure this")
	}

	pos := d.idx % len(d.log)
	s.CopyTo(&d.log[pos])

	d.maxRev = s.Rev

	if d.size == 0 {
		d.minRev = s.Rev - 1
	} else if d.size >= len(d.log) {
		// Buffer is full, wrapping around - oldest is at next position
		oldestPos := (d.idx + 1) % len(d.log)
		d.minRev = d.log[oldestPos].Rev - 1
	}

	d.idx++
	if d.size < len(d.log) {
		d.size++
	}
}

func (d *cacheDriver) Init(e *Engine) error {
	return d.d.Init(e)
}

func (d *cacheDriver) GetStateByID(cmd *GetStateByIDCommand) error {
	if cmd.Rev != 0 && d.getStateByIDFromLog(cmd) {
		return nil
	}

	return d.d.GetStateByID(cmd)
}

func (d *cacheDriver) getStateByIDFromLog(cmd *GetStateByIDCommand) bool {
	d.m.Lock()
	defer d.m.Unlock()

	if d.size == 0 {
		return false
	}
	if cmd.Rev != 0 && cmd.Rev <= d.minRev {
		return false
	}
	if cmd.Rev != 0 && cmd.Rev > d.maxRev {
		return false
	}

	for i := 0; i < d.size; i++ {
		pos := (d.idx - 1 - i + len(d.log)) % len(d.log)
		s := d.log[pos]

		if cmd.ID == s.ID && (cmd.Rev == 0 || cmd.Rev == s.Rev) {
			s.CopyToCtx(cmd.StateCtx)
			return true
		}
	}

	return false
}

func (d *cacheDriver) GetStateByLabels(cmd *GetStateByLabelsCommand) error {
	return d.d.GetStateByLabels(cmd)
}

func (d *cacheDriver) GetStates(cmd *GetStatesCommand) error {
	if _, _, found := d.getStatesFromLog(cmd); found {
		return nil
	}

	if err := d.d.GetStates(cmd); err != nil {
		return err
	}

	return nil
}

func (d *cacheDriver) getStatesFromLog(cmd *GetStatesCommand) (int64, int64, bool) {
	d.m.Lock()
	defer d.m.Unlock()

	if !cmd.SinceTime.IsZero() {
		return d.minRev, d.maxRev, false
	}
	if cmd.LatestOnly {
		return d.minRev, d.maxRev, false
	}

	if d.size == 0 {
		return d.minRev, d.maxRev, false
	}
	if cmd.SinceRev < d.minRev {
		return d.minRev, d.maxRev, false
	}
	if cmd.SinceRev >= d.maxRev {
		return d.minRev, d.maxRev, false
	}

	cmd.Result = &GetStatesResult{}
	for i := 0; i < d.size; i++ {
		pos := (d.idx - d.size + i) % len(d.log)
		s := d.log[pos]
		if s.Rev <= cmd.SinceRev {
			continue
		}
		if len(cmd.Labels) > 0 && !matchLabels(s, cmd.Labels) {
			continue
		}

		// Check if we've hit the limit
		if cmd.Limit > 0 && len(cmd.Result.States) >= cmd.Limit {
			cmd.Result.More = true
			break
		}

		cmd.Result.States = append(cmd.Result.States, s.CopyTo(&State{}))
	}

	return d.minRev, d.maxRev, true
}

func (d *cacheDriver) GetDelayedStates(cmd *GetDelayedStatesCommand) error {
	return d.d.GetDelayedStates(cmd)
}

func (d *cacheDriver) Delay(cmd *DelayCommand) error {
	return d.d.Delay(cmd)
}

func (d *cacheDriver) Commit(cmd *CommitCommand) error {
	if err := d.commitRevMismatch(cmd); err != nil {
		return err
	}

	if err := d.d.Commit(cmd); err != nil {
		return err
	}

	d.commitAppendLog(cmd)

	return nil
}

func (d *cacheDriver) commitRevMismatch(cmd *CommitCommand) error {
	revMismatchErr := &ErrRevMismatch{}
	copyStateCtx := &StateCtx{}
	for _, subCmd0 := range cmd.Commands {
		subCmd, ok := subCmd0.(CommittableCommand)
		if !ok {
			continue
		}

		stateCtx := subCmd.CommittableStateCtx()
		copyStateCtx = stateCtx.CopyTo(copyStateCtx)

		getCmd := GetStateByID(copyStateCtx, stateCtx.Current.ID, 0)
		if !d.getStateByIDFromLog(getCmd) {
			continue
		}

		if copyStateCtx.Current.Rev > stateCtx.Current.Rev {
			revMismatchErr.Add(stateCtx.Current.ID)
		}
	}

	if len(revMismatchErr.IDS) > 0 {
		return revMismatchErr
	}

	return nil
}

func (d *cacheDriver) commitAppendLog(cmd *CommitCommand) {
	d.m.Lock()
	defer d.m.Unlock()

	for _, subCmd0 := range cmd.Commands {
		subCmd, ok := subCmd0.(CommittableCommand)
		if !ok {
			continue
		}

		if len(d.committed) >= cap(d.committed) {
			break
		}

		stateCtx := subCmd.CommittableStateCtx()
		d.committed = append(d.committed, stateCtx.Current.CopyTo(&State{}))
	}

	sort.Slice(d.committed, func(i, j int) bool {
		return d.committed[i].Rev < d.committed[j].Rev
	})

	i := 0
	for i < len(d.committed) {
		s := d.committed[i]
		expRev := d.maxRev + 1
		if s.Rev < expRev {
			i++
			continue
		}
		if expRev == s.Rev {
			d.appendStateLocked(&s)
			i++
			continue
		}
		break
	}

	d.committed = append(d.committed[:0], d.committed[i:]...)
}

func (d *cacheDriver) GetData(cmd *GetDataCommand) error {
	return d.d.GetData(cmd)
}

func (d *cacheDriver) StoreData(cmd *StoreDataCommand) error {
	return d.d.StoreData(cmd)
}

func (d *cacheDriver) getHead(refreshDur, refreshErrDur time.Duration, closeCh chan struct{}) {
	for {
		d.m.Lock()
		maxRev := d.maxRev
		d.m.Unlock()

		if maxRev <= 0 {
			getCmd := GetStatesByLabels(nil).
				WithSinceLatest().
				WithLatestOnly().
				WithLimit(1)
			if err := d.d.GetStates(getCmd); err != nil {
				d.l.Error(fmt.Sprintf("get head: get states failed; retrying in %s", refreshErrDur), "error", err)
				waitT := time.NewTimer(refreshErrDur)
				select {
				case <-waitT.C:
					continue
				case <-closeCh:
					waitT.Stop()
					return
				}
			}

			getRes := getCmd.MustResult()
			if len(getRes.States) == 0 {
				waitT := time.NewTimer(refreshDur)
				select {
				case <-waitT.C:
					continue
				case <-closeCh:
					waitT.Stop()
					return
				}
			}

			d.m.Lock()
			if d.maxRev == 0 || d.maxRev+1 == getRes.States[0].Rev {
				d.appendStateLocked(&getRes.States[0])
			}
			maxRev = d.maxRev
			d.m.Unlock()
			continue
		}

		getCmd := GetStatesByLabels(nil).
			WithSinceRev(maxRev).
			WithLimit(min(100, cap(d.log)))
		if err := d.d.GetStates(getCmd); err != nil {
			d.l.Error(fmt.Sprintf("get head: get states failed; retrying in %s", refreshErrDur), "error", err)
			waitT := time.NewTimer(refreshErrDur)
			select {
			case <-waitT.C:
				continue
			case <-closeCh:
				waitT.Stop()
				return
			}
		}
		getRes := getCmd.MustResult()
		if len(getRes.States) != 0 {
			d.m.Lock()
			d.committed = d.committed[:0]
			for i := range getRes.States {
				if getRes.States[i].Rev <= d.maxRev {
					continue
				}

				d.appendStateLocked(&getRes.States[i])
				maxRev = d.maxRev
			}
			d.m.Unlock()

			if getRes.More {
				continue
			}
		}

		waitT := time.NewTimer(refreshDur)
		select {
		case <-waitT.C:
			continue
		case <-closeCh:
			waitT.Stop()
			return
		}
	}
}

func matchLabels(state State, orLabels []map[string]string) bool {
	if len(orLabels) == 0 {
		return true
	}

next:
	for _, labels := range orLabels {
		if len(labels) == 0 {
			return true
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
