package memdriver

import (
	"slices"
	"sync"

	"github.com/makasim/flowstate"
)

type Log struct {
	sync.Mutex
	rev     int64
	entries []*flowstate.TaskCtx

	changes []*flowstate.TaskCtx
}

func (l *Log) Append(taskCtx *flowstate.TaskCtx) {
	committedT, _ := l.LatestByID(taskCtx.Current.ID)
	if committedT == nil {
		committedT = &flowstate.TaskCtx{}
	}

	taskCtx.CopyTo(committedT)
	committedT.Current.CopyTo(&committedT.Committed)
	committedT.Transitions = committedT.Transitions[:0]

	l.rev++
	committedT.Committed.Rev = l.rev
	committedT.Current.Rev = l.rev

	l.changes = append(l.changes, committedT)

	// todo: find a better place for this
	committedT.Committed.CopyTo(&taskCtx.Current)
	committedT.Committed.CopyTo(&taskCtx.Committed)
	taskCtx.Transitions = taskCtx.Transitions[:0]
}

func (l *Log) Commit() {
	slices.CompactFunc(l.changes, func(l, r *flowstate.TaskCtx) bool {
		return l.Committed.ID == r.Committed.ID
	})

	slices.SortFunc(l.changes, func(l, r *flowstate.TaskCtx) int {
		if l.Committed.Rev < r.Committed.Rev {
			return -1
		}

		return 1
	})

	for _, taskCtx := range l.changes {
		l.entries = append(l.entries, taskCtx)
	}

	l.changes = l.changes[:0]
}

func (l *Log) Rollback() {
	l.changes = l.changes[:0]
}

func (l *Log) LatestByID(tID flowstate.TaskID) (*flowstate.TaskCtx, int64) {
	var since int64
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == tID {
			since = l.entries[i].Committed.Rev
			return l.entries[i].CopyTo(&flowstate.TaskCtx{}), since
		}
	}

	return nil, since
}

func (l *Log) LatestByLabels(labels map[string]string) (*flowstate.TaskCtx, int64) {
	var since int64
next:
	for i := len(l.entries) - 1; i >= 0; i-- {
		for k, v := range labels {
			if l.entries[i].Committed.Labels[k] != v {
				continue next
			}
		}

		since = l.entries[i].Committed.Rev
		return l.entries[i].CopyTo(&flowstate.TaskCtx{}), since
	}

	return nil, since
}

func (l *Log) Entries(since int64, limit int) ([]*flowstate.TaskCtx, int64) {
	if limit == 0 {
		return nil, since
	}

	var entries []*flowstate.TaskCtx
	for i := 0; i < len(l.entries); i++ {
		if l.entries[i].Committed.Rev <= since {
			continue
		}

		to := l.entries[i].CopyTo(&flowstate.TaskCtx{})
		since = to.Committed.Rev

		entries = append(entries, to)
		if len(entries) == limit {
			break
		}
	}

	return entries, since
}
