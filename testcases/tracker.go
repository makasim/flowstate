package testcases

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Tracker struct {
	IncludeTaskID bool
	IncludeState  bool

	mux     sync.Mutex
	visited []string
}

func Track(stateCtx *flowstate.StateCtx, trkr *Tracker) {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	var postfix string

	if trkr.IncludeState {
		switch {
		case flowstate.Resumed(stateCtx.Current):
			postfix += `:resumed`
		case flowstate.Paused(stateCtx.Current):
			postfix += `:paused`
		}
	}

	if trkr.IncludeTaskID {
		postfix += `:` + string(stateCtx.Current.ID)
	}

	trkr.visited = append(trkr.visited, string(stateCtx.Current.Transition.To)+postfix)
}

func (trkr *Tracker) Visited() []string {
	trkr.mux.Lock()
	defer trkr.mux.Unlock()

	return append([]string(nil), trkr.visited...)
}

func (trkr *Tracker) VisitedSorted() []string {
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

func (trkr *Tracker) WaitSortedVisitedEqual(t *testing.T, expVisited []string, wait time.Duration) []string {
	t.Helper()

	var visited []string
	assert.Eventually(t, func() bool {
		visited = trkr.VisitedSorted()
		return len(visited) >= len(expVisited)
	}, wait, time.Millisecond*50)
	require.Equal(t, expVisited, visited)

	return visited
}

func (trkr *Tracker) WaitVisitedEqual(t *testing.T, expVisited []string, wait time.Duration) []string {
	t.Helper()

	var visited []string
	assert.Eventually(t, func() bool {
		visited = trkr.Visited()
		return len(visited) >= len(expVisited)
	}, wait, time.Millisecond*50)
	require.Equal(t, expVisited, visited)

	return visited
}

func mustSetFlow(fr flowstate.FlowRegistry, id flowstate.TransitionID, f flowstate.Flow) {
	if err := fr.SetFlow(id, f); err != nil {
		panic(fmt.Sprintf("set flow %s: %s", id, err))
	}
}
