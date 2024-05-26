package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestFork_WithCommit(t *testing.T) {
	var forkedStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &tracker2{}

	fr := memdriver.NewFlowRegistry()
	fr.SetFlow("fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

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

		return flowstate.Nop(stateCtx), nil
	}))
	fr.SetFlow("forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	fr.SetFlow("origin", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, fr)

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `fork`)))
	require.NoError(t, e.Execute(stateCtx))

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []string{
		`fork`,
		`forked`,
		`origin`,
	}, trkr.VisitedSorted())
}
