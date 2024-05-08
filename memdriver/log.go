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
	committedT, _ := l.Latest(taskCtx.Current.ID)
	if committedT == nil {
		committedT = &flowstate.TaskCtx{}
	}

	taskCtx.CopyTo(committedT)
	committedT.Current.CopyTo(&committedT.Committed)
	committedT.Transitions = committedT.Transitions[:0]

	l.rev++
	committedT.Committed.Rev = l.rev

	l.changes = append(l.changes, committedT)

	committedT.Committed.CopyTo(&taskCtx.Current)
	committedT.Committed.CopyTo(&taskCtx.Committed)
	taskCtx.Transitions = taskCtx.Transitions[:0]
}

func (l *Log) Commit() {
	slices.CompactFunc(l.changes, func(leftTaskCtx, rightTaskCtx *flowstate.TaskCtx) bool {
		return leftTaskCtx.Committed.ID == rightTaskCtx.Committed.ID
	})

	slices.SortFunc(l.changes, func(leftTaskCtx, rightTaskCtx *flowstate.TaskCtx) int {
		if leftTaskCtx.Committed.Rev < rightTaskCtx.Committed.Rev {
			return -1
		}

		return 1
	})

	for _, taskCtx := range l.changes {
		l.entries = append(l.entries, taskCtx)
	}
}

func (l *Log) Rollback() {
	l.changes = l.changes[:0]
}

func (l *Log) Latest(tID flowstate.TaskID) (*flowstate.TaskCtx, int64) {
	var since int64
	for i := len(l.entries) - 1; i >= 0; i-- {
		if l.entries[i].Committed.ID == tID {
			since = l.entries[i].Committed.Rev
			return l.entries[i].CopyTo(&flowstate.TaskCtx{}), since
		}
	}

	return nil, since
}
