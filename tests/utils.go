package tests

import (
	"sort"
	"sync"

	"github.com/makasim/flowstate"
)

type tracker struct {
	IncludeTaskID bool
	IncludeState  bool

	mux     sync.Mutex
	visited []flowstate.TransitionID
}

func track(taskCtx *flowstate.TaskCtx, trkr *tracker) {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	var postfix string

	if trkr.IncludeState {
		switch {
		case flowstate.Resumed(taskCtx):
			postfix += `:resumed`
		}
	}

	if trkr.IncludeTaskID {
		postfix += `:` + string(taskCtx.Current.ID)
	}

	trkr.visited = append(trkr.visited, taskCtx.Current.Transition.ID+flowstate.TransitionID(postfix))
}

func (trkr *tracker) Visited() []flowstate.TransitionID {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	return append([]flowstate.TransitionID(nil), trkr.visited...)
}

func (trkr *tracker) VisitedSorted() []flowstate.TransitionID {
	visited := trkr.Visited()

	// sort to eliminate race conditions
	sort.SliceStable(visited, func(i, j int) bool {
		if string(visited[i]) > string(visited[j]) {
			return false
		}
		return true
	})

	return visited
}
