package tests

import (
	"sort"
	"sync"

	"github.com/makasim/flowstate"
)

type tracker struct {
	mux     sync.Mutex
	visited []flowstate.TransitionID
}

func track(taskCtx *flowstate.TaskCtx, trkr *tracker) {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	trkr.visited = append(trkr.visited, taskCtx.Current.Transition.ID)
}

func (trkr *tracker) Visited() []flowstate.TransitionID {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	return append([]flowstate.TransitionID(nil), trkr.visited...)
}

func (trkr *tracker) VisitedSorted() []flowstate.TransitionID {
	visited := trkr.Visited()

	// sort to eliminate race conditions
	sort.Slice(visited, func(i, j int) bool {
		if string(trkr.Visited()[i]) > string(trkr.Visited()[j]) {
			return false
		}
		return true
	})

	return visited
}
