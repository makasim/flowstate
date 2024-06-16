package memdriver

import (
	"fmt"
	"slices"
	"sync"

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
	committedT, _ := l.LatestByID(stateCtx.Current.ID)
	if committedT == nil {
		committedT = &flowstate.StateCtx{}
	}

	stateCtx.CopyTo(committedT)
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

func (l *Log) LatestByID(tID flowstate.StateID) (*flowstate.StateCtx, int64) {
	var since int64
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == tID {
			since = l.entries[i].Committed.Rev
			return l.entries[i].CopyTo(&flowstate.StateCtx{}), since
		}
	}

	return nil, since
}

func (l *Log) LatestByLabels(labels map[string]string) (*flowstate.StateCtx, int64) {
	var since int64
next:
	for i := len(l.entries) - 1; i >= 0; i-- {
		for k, v := range labels {
			if l.entries[i].Committed.Labels[k] != v {
				continue next
			}
		}

		since = l.entries[i].Committed.Rev
		return l.entries[i].CopyTo(&flowstate.StateCtx{}), since
	}

	return nil, since
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

	//log.Println(91)
	l.listeners = append(l.listeners, notifyCh)
	return nil
}

func (l *Log) UnsubscribeCommit(notifyCh chan int64) {
	l.Lock()
	defer l.Unlock()
	//log.Println(92)
	for i, ch := range l.listeners {
		if ch == notifyCh {
			l.listeners = append(l.listeners[:i], l.listeners[i+1:]...)
			return
		}
	}
}
