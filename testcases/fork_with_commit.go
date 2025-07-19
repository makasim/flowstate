package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Fork_WithCommit(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	var forkedStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	mustSetFlow(fr, "fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		forkedStateCtx = &flowstate.StateCtx{}
		stateCtx.CopyTo(forkedStateCtx)
		forkedStateCtx.Current.ID = "forkedTID"
		forkedStateCtx.Current.Rev = 0
		forkedStateCtx.Current.CopyTo(&forkedStateCtx.Committed)

		if err := e.Do(
			flowstate.Commit(
				flowstate.Transit(stateCtx, `origin`),
				flowstate.Transit(forkedStateCtx, `forked`),
			),
			flowstate.Execute(stateCtx),
			flowstate.Execute(forkedStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	mustSetFlow(fr, "forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	mustSetFlow(fr, "origin", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `fork`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitSortedVisitedEqual(t, []string{
		`fork`,
		`forked`,
		`origin`,
	}, time.Second)
}
