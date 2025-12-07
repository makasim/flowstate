package flowstate

import (
	"log"
	"log/slog"
	"sort"
	"sync"
	"time"
)

var _ Driver = &cacheDriver{}

// A cacheDriver is a driver that keeps a head of states and updates them by worker from time to time.
// The head refresh can also be triggered by state commit commands.
// The driver clients can subscribe to head updates.
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

	closeCh  chan struct{}
	closedCh chan struct{}
}

func newCacheDriver(d Driver, maxSize int, l *slog.Logger) *cacheDriver {
	hd := &cacheDriver{
		d: d,
		l: l,

		log:       make([]State, maxSize),
		committed: make([]State, 0, 100),
		size:      0,
		idx:       0,
		minRev:    -1,
		maxRev:    -1,
		closeCh:   make(chan struct{}),
		closedCh:  make(chan struct{}),
	}

	//go hd.getHead()

	return hd
}

func (d *cacheDriver) close() {
	close(d.closeCh)
	<-d.closedCh
}

// appendStateLocked appends a state to the cycle buffer log.
// Must be called with d.m held.
func (d *cacheDriver) appendStateLocked(s *State) {
	pos := d.idx % len(d.log)
	s.CopyTo(&d.log[pos])

	d.maxRev = s.Rev

	if d.size == 0 {
		d.minRev = s.Rev
	} else if d.size >= len(d.log) {
		// Buffer is full, wrapping around - oldest is at next position
		oldestPos := (d.idx + 1) % len(d.log)
		d.minRev = d.log[oldestPos].Rev
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
	if d.getStateByIDFromLog(cmd) {
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
	if cmd.Rev != 0 && cmd.Rev < d.minRev {
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
	if d.getStateByLabelsFromLog(cmd) {
		return nil
	}

	return d.d.GetStateByLabels(cmd)
}

func (d *cacheDriver) getStateByLabelsFromLog(cmd *GetStateByLabelsCommand) bool {
	d.m.Lock()
	defer d.m.Unlock()

	if d.size == 0 {
		return false
	}

	if len(cmd.Labels) == 0 {
		// return last state from the log
		pos := (d.idx - 1 + len(d.log)) % len(d.log)
		s := d.log[pos]
		s.CopyToCtx(cmd.StateCtx)
		return true
	}

	orLabels := []map[string]string{cmd.Labels}

	for i := 0; i < d.size; i++ {
		pos := (d.idx - 1 - i + len(d.log)) % len(d.log)
		s := d.log[pos]

		if !matchLabels(s, orLabels) {
			continue
		}

		s.CopyToCtx(cmd.StateCtx)
		return true
	}

	return false
}

func (d *cacheDriver) GetStates(cmd *GetStatesCommand) error {
	if d.getStatesFromLog(cmd) {
		return nil
	}

	return d.d.GetStates(cmd)
}

func (d *cacheDriver) getStatesFromLog(cmd *GetStatesCommand) bool {
	d.m.Lock()
	defer d.m.Unlock()

	if d.size == 0 {
		return false
	}
	if cmd.SinceRev < d.minRev {
		return false
	}
	if cmd.SinceRev > d.maxRev {
		return false
	}

	cmd.Result = &GetStatesResult{}
	for i := 0; i < d.size; i++ {
		pos := (d.idx - d.size + i) % len(d.log)
		s := d.log[pos]
		if s.Rev < cmd.SinceRev {
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

	return true
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

		if copyStateCtx.Committed.Rev > stateCtx.Current.Rev {
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

	committed := d.committed
	for _, s := range d.committed {
		if d.maxRev+1 == s.Rev {
			d.appendStateLocked(&s)
			committed = committed[1:]
			continue
		}

		break
	}
	d.committed = append(d.committed[:0], committed...)
}

func (d *cacheDriver) GetData(cmd *GetDataCommand) error {
	return d.d.GetData(cmd)
}

func (d *cacheDriver) StoreData(cmd *StoreDataCommand) error {
	return d.d.StoreData(cmd)
}

func (d *cacheDriver) getHead() {
	defer close(d.closedCh)

	headRev := int64(-1)

	for {
		if headRev < 0 {
			getCmd := GetStatesByLabels(nil).
				WithSinceLatest().
				WithLatestOnly().
				WithLimit(1)
			if err := d.d.GetStates(getCmd); err != nil {
				d.l.Error("get head: get states failed; retrying in 5s", "error", err)
				waitT := time.NewTimer(time.Second * 5)
				select {
				case <-waitT.C:
					continue
				case <-d.closeCh:
					return
				}
			}

			getRes := getCmd.MustResult()
			if len(getRes.States) == 0 {
				waitT := time.NewTimer(time.Second)
				select {
				case <-waitT.C:
					continue
				case <-d.closeCh:
					return
				}
			}

			headRev = getRes.States[0].Rev
			d.m.Lock()
			d.appendStateLocked(&getRes.States[0])
			d.m.Unlock()
			continue
		}

		getCmd := GetStatesByLabels(nil).
			WithSinceRev(headRev).
			WithLatestOnly().
			WithLimit(min(1000, cap(d.log)))
		if err := d.d.GetStates(getCmd); err != nil {
			d.l.Error("get head: get states failed; retrying in 5s", "error", err)
			waitT := time.NewTimer(time.Second * 5)
			select {
			case <-waitT.C:
				continue
			case <-d.closeCh:
				return
			}
		}
		getRes := getCmd.MustResult()
		if len(getRes.States) != 0 {
			d.m.Lock()
			d.committed = d.committed[:0]
			for i := range getRes.States {
				d.appendStateLocked(&getRes.States[i])
				headRev = getRes.States[i].Rev
			}
			d.m.Unlock()

			if getRes.More {
				continue
			}
		}

		waitT := time.NewTimer(time.Second)
		select {
		case <-waitT.C:
			continue
		case <-d.closeCh:
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
