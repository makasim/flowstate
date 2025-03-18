package memdriver

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/makasim/flowstate"
)

type Log struct {
	sync.Mutex
	rev     int64
	entries []*flowstate.StateCtx

	changes []*flowstate.StateCtx

	listeners []chan int64
}

func (l *Log) Append(stateCtx *flowstate.StateCtx) {
	committedT, _ := l.GetLatestByID(stateCtx.Current.ID)
	if committedT == nil {
		committedT = &flowstate.StateCtx{}
	}

	stateCtx.CopyTo(committedT)
	committedT.Current.SetCommitedAt(time.Now())
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

func (l *Log) Commit() {
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

func (l *Log) Rollback() {
	l.changes = l.changes[:0]
}

func (l *Log) GetLatestByID(id flowstate.StateID) (*flowstate.StateCtx, int64) {
	var since int64
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == id {
			since = l.entries[i].Committed.Rev
			return l.entries[i].CopyTo(&flowstate.StateCtx{}), since
		}
	}

	return nil, since
}

func (l *Log) GetByIDAndRev(id flowstate.StateID, rev int64) *flowstate.StateCtx {
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == id && l.entries[i].Committed.Rev == rev {
			return l.entries[i].CopyTo(&flowstate.StateCtx{})
		}
	}

	return nil
}

func (l *Log) GetLatestByLabels(orLabels []map[string]string) (*flowstate.StateCtx, int64) {
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

func (l *Log) Entries(since int64, limit int) ([]*flowstate.StateCtx, int64) {
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

func (l *Log) SubscribeCommit(notifyCh chan int64) error {
	if cap(notifyCh) == 0 {
		return fmt.Errorf("notify channel is not buffered")
	}

	l.Lock()
	defer l.Unlock()

	l.listeners = append(l.listeners, notifyCh)
	return nil
}

func (l *Log) UnsubscribeCommit(notifyCh chan int64) {
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
