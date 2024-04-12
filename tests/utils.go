package tests

import (
	"sync"

	"github.com/makasim/flowstate"
)

type tracker struct {
	mux     sync.Mutex
	visited []flowstate.TransitionID
}

func track(taskCtx *flowstate.TaskCtx, trkr *tracker) {
	trkr.visited = append(trkr.visited, taskCtx.Current.Transition.ID)
}

type nopDriver struct {
	calls int
}

func (d *nopDriver) Commit(_ ...flowstate.Command) error {
	d.calls++
	return nil
}
