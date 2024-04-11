package tests

import (
	"sort"
	"sync"

	"github.com/makasim/flowstate"
)

type tracker2 struct {
	IncludeTaskID bool
	IncludeState  bool

	mux     sync.Mutex
	visited []string
}

func track2(stateCtx *flowstate.StateCtx, trkr *tracker2) {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	var postfix string

	if trkr.IncludeState {
		switch {
		case flowstate.Resumed(stateCtx):
			postfix += `:resumed`
		case flowstate.Paused(stateCtx):
			postfix += `:paused`
		}
	}

	if trkr.IncludeTaskID {
		postfix += `:` + string(stateCtx.Current.ID)
	}

	trkr.visited = append(trkr.visited, string(stateCtx.Current.Transition.ToID)+postfix)
}

func (trkr *tracker2) Visited() []string {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	return append([]string(nil), trkr.visited...)
}

func (trkr *tracker2) VisitedSorted() []string {
	visited := trkr.Visited()

	// sort to eliminate race conditions
	sort.SliceStable(visited, func(i, j int) bool {
		if visited[i] > visited[j] {
			return false
		}
		return true
	})

	return visited
}
